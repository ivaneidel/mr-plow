package scan

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Options configure a scan.
type Options struct {
	Root    string
	Lister  Lister // nil = DefaultLister()
	Workers int    // concurrent directory listings; 0 = 4×NumCPU
	BigMin  int64  // files at least this big go to the big-file list (0 = 50 MB)
	BigKeep int    // how many big files to keep (0 = 500)
	// Detect tags a child directory given its name and its siblings
	// (the parent's entries). Return "" for no tag.
	Detect func(siblings []Entry, child string) string
	// Skip prunes absolute paths (not descended, flagged FlagSkipped).
	Skip func(path string) bool
	// AllowDevs are extra devices that count as "same filesystem".
	AllowDevs []uint64
	// Prioritize marks paths whose subtrees should be scanned first
	// (e.g. ancestors of known cache locations), so they finish early.
	Prioritize func(path string) bool
}

// Progress is a snapshot of a running scan.
type Progress struct {
	Dirs, Files, Bytes int64
	Denied, Errors     int64
	Current            string
	Elapsed            time.Duration
	Done               bool
}

// Scanner runs one scan. Create with New, then Start and Wait.
type Scanner struct {
	opts Options
	list Lister
	root *Dir
	devs map[uint64]bool
	wg   sync.WaitGroup

	// work queue: a LIFO stack shared by all workers (depth-first keeps it small)
	qmu      sync.Mutex
	cond     *sync.Cond
	queue    []job
	inflight int // queued + being listed
	links    inodeSet
	big      topFiles

	dirs, files, bytes, denied, errs atomic.Int64
	current                          atomic.Pointer[string]
	stop, finished                   atomic.Bool
	start                            time.Time
	elapsed                          atomic.Int64

	mu        sync.Mutex
	tagged    []*Dir
	deniedSet []string
	doneCh    chan struct{}
}

// New prepares a scan of opts.Root (symlinks in the root path are resolved).
func New(opts Options) (*Scanner, error) {
	root, err := filepath.Abs(opts.Root)
	if err != nil {
		return nil, err
	}
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	var st syscall.Stat_t
	if err := syscall.Stat(root, &st); err != nil {
		return nil, err
	}
	if st.Mode&syscall.S_IFMT != syscall.S_IFDIR {
		return nil, errors.New(root + " is not a directory")
	}
	if opts.Lister == nil {
		opts.Lister = DefaultLister()
	}
	if opts.Workers <= 0 {
		opts.Workers = 4 * runtime.NumCPU()
	}
	if opts.BigMin <= 0 {
		opts.BigMin = 50 << 20
	}
	if opts.BigKeep <= 0 {
		opts.BigKeep = 500
	}
	s := &Scanner{
		opts:   opts,
		list:   opts.Lister,
		devs:   map[uint64]bool{uint64(st.Dev): true},
		doneCh: make(chan struct{}),
	}
	for _, d := range opts.AllowDevs {
		s.devs[d] = true
	}
	s.big.keep = opts.BigKeep
	s.root = &Dir{Name: root, Dev: uint64(st.Dev), Ino: uint64(st.Ino)}
	return s, nil
}

func (s *Scanner) Root() *Dir     { return s.root }
func (s *Scanner) Lister() string { return s.list.Name() }

// Start begins scanning in the background.
func (s *Scanner) Start() {
	s.start = time.Now()
	s.root.pending.Store(1)
	s.queue = []job{{s.root, s.root.Name}}
	s.inflight = 1
	s.cond = sync.NewCond(&s.qmu)
	for i := 0; i < s.opts.Workers; i++ {
		s.wg.Add(1)
		go s.worker()
	}
	go func() {
		s.wg.Wait()
		s.root.finalize()
		s.elapsed.Store(int64(time.Since(s.start)))
		s.finished.Store(true)
		close(s.doneCh)
	}()
}

// Stop asks the workers to stop early. Results stay partial.
func (s *Scanner) Stop() { s.stop.Store(true) }

// Stopped reports whether Stop was called (results are partial).
func (s *Scanner) Stopped() bool { return s.stop.Load() }

// Wait blocks until the scan finishes.
func (s *Scanner) Wait() { <-s.doneCh }

// DoneCh is closed when the scan finishes.
func (s *Scanner) DoneCh() <-chan struct{} { return s.doneCh }

// Progress returns live counters.
func (s *Scanner) Progress() Progress {
	p := Progress{
		Dirs: s.dirs.Load(), Files: s.files.Load(), Bytes: s.bytes.Load(),
		Denied: s.denied.Load(), Errors: s.errs.Load(),
		Done: s.finished.Load(),
	}
	if c := s.current.Load(); c != nil {
		p.Current = *c
	}
	if p.Done {
		p.Elapsed = time.Duration(s.elapsed.Load())
	} else if !s.start.IsZero() {
		p.Elapsed = time.Since(s.start)
	}
	return p
}

// Tagged returns the directories the detector tagged so far.
func (s *Scanner) Tagged() []*Dir {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*Dir(nil), s.tagged...)
}

// DeniedPaths returns up to 200 directories that could not be read.
func (s *Scanner) DeniedPaths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.deniedSet...)
}

// BigFiles returns the biggest files, largest first.
func (s *Scanner) BigFiles() []File { return s.big.sorted() }

func (s *Scanner) walk(d *Dir, path string) {
	if s.stop.Load() {
		s.markDone(d)
		return
	}
	s.current.Store(&path)
	ents, err := s.list.List(path, nil)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) || errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			d.SetFlag(FlagDenied)
			s.denied.Add(1)
			s.mu.Lock()
			if len(s.deniedSet) < 200 {
				s.deniedSet = append(s.deniedSet, path)
			}
			s.mu.Unlock()
		} else if !errors.Is(err, os.ErrNotExist) {
			s.errs.Add(1)
		}
	}

	var selfSize, selfFiles, newest int64
	var subdirs []*Dir
	childTagged := d.inTagged || d.Tag != ""
	for i := range ents {
		e := &ents[i]
		if e.MTime > newest {
			newest = e.MTime
		}
		switch e.Kind {
		case KindDir:
			c := &Dir{Name: e.Name, Parent: d, Dev: e.Dev, Ino: e.Ino, inTagged: childTagged}
			if e.Mount || !s.devs[e.Dev] {
				c.SetFlag(FlagMount)
				c.done.Store(true)
			} else if s.opts.Skip != nil && s.opts.Skip(filepath.Join(path, e.Name)) {
				c.SetFlag(FlagSkipped)
				c.done.Store(true)
			} else if !childTagged && s.opts.Detect != nil {
				c.Tag = s.opts.Detect(ents, e.Name)
			}
			subdirs = append(subdirs, c)
		default:
			alloc := e.Alloc
			if e.Kind == KindFile && e.Nlink > 1 && !s.links.firstSeen(e.Dev, e.Ino) {
				alloc = 0 // hard link already counted elsewhere
			}
			selfSize += alloc
			selfFiles++
			if e.Kind == KindFile && alloc >= s.opts.BigMin {
				s.big.offer(File{Path: filepath.Join(path, e.Name), Size: alloc, MTime: e.MTime, Dev: e.Dev, Ino: e.Ino})
			}
		}
	}
	d.selfNewest = newest
	for _, c := range subdirs {
		if c.Tag != "" {
			s.mu.Lock()
			s.tagged = append(s.tagged, c)
			s.mu.Unlock()
		}
	}
	for p := d; p != nil; p = p.Parent {
		p.size.Add(selfSize)
		p.files.Add(selfFiles)
	}
	s.dirs.Add(1)
	s.files.Add(selfFiles)
	s.bytes.Add(selfSize)

	var todo []*Dir
	for _, c := range subdirs {
		if !c.Done() {
			todo = append(todo, c)
		}
	}
	d.pending.Add(int32(len(todo)))
	d.kids.Store(&subdirs) // publish only after children are fully initialised

	if len(todo) > 0 {
		s.qmu.Lock()
		// LIFO: whatever is pushed last is listed first, so priority goes last.
		var prio []job
		for _, c := range todo {
			c.pending.Store(1)
			j := job{c, filepath.Join(path, c.Name)}
			if s.opts.Prioritize != nil && s.opts.Prioritize(j.path) {
				prio = append(prio, j)
			} else {
				s.queue = append(s.queue, j)
			}
		}
		s.queue = append(s.queue, prio...)
		s.inflight += len(todo)
		s.qmu.Unlock()
		if len(todo) == 1 {
			s.cond.Signal()
		} else {
			s.cond.Broadcast()
		}
	}
	s.markDone(d)
}

type job struct {
	d    *Dir
	path string
}

func (s *Scanner) worker() {
	defer s.wg.Done()
	for {
		s.qmu.Lock()
		for len(s.queue) == 0 && s.inflight > 0 {
			s.cond.Wait()
		}
		if s.inflight == 0 {
			s.qmu.Unlock()
			s.cond.Broadcast()
			return
		}
		j := s.queue[len(s.queue)-1]
		s.queue = s.queue[:len(s.queue)-1]
		s.qmu.Unlock()

		s.walk(j.d, j.path)

		s.qmu.Lock()
		s.inflight--
		last := s.inflight == 0
		s.qmu.Unlock()
		if last {
			s.cond.Broadcast()
		}
	}
}

// markDone drops d's own pending count and bubbles completion upwards.
func (s *Scanner) markDone(d *Dir) {
	for p := d; p != nil; p = p.Parent {
		if p.pending.Add(-1) != 0 {
			return
		}
		p.done.Store(true)
	}
}
