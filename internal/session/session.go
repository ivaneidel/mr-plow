// Package session ties a scan, the catalog and the cache together into what
// the UI shows.
package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ivaneidel/mr-plow/internal/catalog"
	"github.com/ivaneidel/mr-plow/internal/disk"
	"github.com/ivaneidel/mr-plow/internal/safety"
	"github.com/ivaneidel/mr-plow/internal/scan"
)

// Options configure a session.
type Options struct {
	Root     string // "" = home
	NoCache  bool
	NoSystem bool // skip system-wide locations and Time Machine (tests)
	Lister   scan.Lister
	Identity safety.Identity
}

// Session owns the running scan(s) and the latest results.
type Session struct {
	Home, Root string
	ID         safety.Identity
	IsHome     bool // scanning the whole home (system extras + cache enabled)

	opts    Options
	entries []catalog.Entry

	mu      sync.Mutex
	scanner *scan.Scanner
	cached  *scan.Snapshot // shown until the fresh scan finishes
	final   *scan.Snapshot // fresh scan, once done
	extras  map[string]*scan.Scanner
	saved   bool

	tmSnapshots atomic.Int32
}

// New prepares a session; call Start to begin scanning.
func New(opts Options) (*Session, error) {
	s := &Session{ID: opts.Identity, Home: opts.Identity.Home, opts: opts, extras: map[string]*scan.Scanner{}}
	root := opts.Root
	if root == "" {
		root = s.Home
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if r, err := filepath.EvalSymlinks(abs); err == nil {
		abs = r
	}
	s.Root = abs
	// scan paths are symlink-free, so home must be too (e.g. HOME=/tmp/… on macOS)
	if r, err := filepath.EvalSymlinks(s.Home); err == nil {
		s.Home = r
	}
	s.IsHome = abs == s.Home
	s.entries = catalog.For(runtime.GOOS)
	return s, nil
}

// Start loads the cache (if any) and launches the scans.
func (s *Session) Start() error { return s.start(true) }

func (s *Session) start(loadCache bool) error {
	if loadCache && s.IsHome && !s.opts.NoCache {
		if sn, err := s.loadCache(); err == nil {
			s.cached = sn
		}
	}
	sc, err := scan.New(scan.Options{
		Root:       s.Root,
		Lister:     s.opts.Lister,
		Detect:     catalog.Detect(),
		Prioritize: s.prioritizer(),
		Skip:       skipper(s.Root),
		AllowDevs:  allowDevs(s.Root),
	})
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.scanner = sc
	s.mu.Unlock()
	sc.Start()
	if s.IsHome && !s.opts.NoSystem {
		for _, e := range s.entries {
			if !e.System || e.Path == "" {
				continue
			}
			for _, p := range e.Resolve(s.Home) {
				if _, err := os.Stat(p); err != nil {
					continue
				}
				if x, err := scan.New(scan.Options{Root: p, Lister: s.opts.Lister}); err == nil {
					s.mu.Lock()
					s.extras[p] = x
					s.mu.Unlock()
					x.Start()
				}
			}
		}
		if runtime.GOOS == "darwin" {
			go s.countSnapshots()
		}
	}
	go func() {
		sc.Wait()
		if sc.Stopped() {
			return // partial: keep showing what we had
		}
		snap := sc.Snapshot()
		s.mu.Lock()
		if s.scanner == sc {
			s.final = snap
		}
		s.mu.Unlock()
		if s.IsHome && !s.opts.NoCache {
			s.saveCache(snap)
		}
	}()
	return nil
}

// Stop aborts running scans.
func (s *Session) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scanner != nil {
		s.scanner.Stop()
	}
	for _, x := range s.extras {
		x.Stop()
	}
}

// WaitAll blocks until the main and system scans finish.
func (s *Session) WaitAll() {
	sc := s.Scanner()
	sc.Wait()
	s.mu.Lock()
	extras := s.extras
	s.mu.Unlock()
	for _, x := range extras {
		x.Wait()
	}
	for !sc.Stopped() { // the final snapshot is published right after Wait
		s.mu.Lock()
		ok := s.final != nil
		s.mu.Unlock()
		if ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Scanner is the current main scanner.
func (s *Session) Scanner() *scan.Scanner {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scanner
}

// Current returns the snapshot to display: the fresh one when done, the cache
// while scanning (if present), otherwise the live partial scan.
func (s *Session) Current() *scan.Snapshot {
	s.mu.Lock()
	final, cached, sc := s.final, s.cached, s.scanner
	s.mu.Unlock()
	switch {
	case final != nil:
		return final
	case cached != nil:
		return cached
	}
	return sc.Snapshot()
}

func (s *Session) extraSnaps() map[string]*scan.Snapshot {
	s.mu.Lock()
	extras := s.extras
	s.mu.Unlock()
	out := map[string]*scan.Snapshot{}
	for p, x := range extras {
		out[p] = x.Snapshot()
	}
	return out
}

// View is everything the UI renders for one frame.
type View struct {
	Snap        *scan.Snapshot
	Live        scan.Progress // the fresh scan's progress
	Groups      []GroupView
	Disk        disk.Usage
	Reclaimable int64 // Safe + Regenerable, de-duplicated
}

// View builds the current view (cheap enough to call every few hundred ms).
func (s *Session) View() View {
	sn := s.Current()
	v := View{Snap: sn, Live: s.Scanner().Progress()}
	v.Groups = s.buildItems(sn)
	v.Disk, _ = disk.Of(s.Root)
	var easy []*Item
	for _, g := range v.Groups {
		for _, it := range g.Items {
			if it.Mode != catalog.ModeInfo && (it.Safety == catalog.Safe || it.Safety == catalog.Regenerable) {
				easy = append(easy, it)
			}
		}
	}
	v.Reclaimable = UnionSize(easy)
	return v
}

// Forget detaches a removed directory or file from every tree we show, so
// sizes update without a rescan.
func (s *Session) Forget(it *Item) {
	switch {
	case it.Node != nil:
		it.Node.Detach()
	case it.File != nil:
		if it.FileDir != nil {
			it.FileDir.SubtractFile(it.File.Size)
		}
		s.dropBig(it.File.Path)
	}
}

func (s *Session) dropBig(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sn := range []*scan.Snapshot{s.final, s.cached} {
		if sn == nil {
			continue
		}
		out := sn.Big[:0]
		for _, f := range sn.Big {
			if f.Path != path {
				out = append(out, f)
			}
		}
		sn.Big = out
	}
}

// Rescan scans again. What is on screen now stays visible (as "last scan")
// until the fresh scan finishes.
func (s *Session) Rescan() error {
	cur := s.Current()
	s.Stop()
	s.Scanner().Wait()
	s.mu.Lock()
	if cur != nil && (cur.Cached || cur.Progress.Done) {
		c := *cur
		c.Cached = true
		s.cached = &c
	}
	s.final = nil
	s.extras = map[string]*scan.Scanner{}
	s.mu.Unlock()
	return s.start(false)
}

// ScanDone reports whether the main scan finished on its own.
func (s *Session) ScanDone() bool {
	sc := s.Scanner()
	return sc.Progress().Done && !sc.Stopped()
}

// SaveCache rewrites the cache from the finished scan (e.g. after deletions).
func (s *Session) SaveCache() {
	s.mu.Lock()
	sn := s.final
	s.mu.Unlock()
	if sn != nil && s.IsHome && !s.opts.NoCache {
		s.saveCache(sn)
	}
}

// prioritizer scans ancestors of known locations first.
func (s *Session) prioritizer() func(string) bool {
	var targets []string
	for _, e := range s.entries {
		if e.System || e.Path == "" {
			continue
		}
		p := e.Path
		if strings.HasPrefix(p, "~/") {
			p = filepath.Join(s.Home, p[2:])
		}
		if i := strings.IndexAny(p, "*?["); i >= 0 {
			p = filepath.Dir(p[:i])
		}
		targets = append(targets, p)
	}
	return func(path string) bool {
		for _, t := range targets {
			if under(t, path) || under(path, t) {
				return true
			}
		}
		return false
	}
}

// skipper avoids double counting through macOS firmlinks when scanning "/".
func skipper(root string) func(string) bool {
	if runtime.GOOS != "darwin" || root != "/" {
		return nil
	}
	return func(p string) bool { return p == "/System/Volumes/Data" }
}

func allowDevs(root string) []uint64 {
	if runtime.GOOS != "darwin" || root != "/" {
		return nil
	}
	var st scanStat
	if statDev("/System/Volumes/Data", &st) == nil {
		return []uint64{st.dev}
	}
	return nil
}

func (s *Session) countSnapshots() {
	out, err := exec.Command("tmutil", "listlocalsnapshots", "/").Output()
	if err != nil {
		return
	}
	n := strings.Count(string(out), "com.apple.TimeMachine.")
	s.tmSnapshots.Store(int32(n))
}

// Age formats "untouched since" for an mtime.
func Age(unix int64) string {
	if unix <= 0 {
		return ""
	}
	t := time.Unix(unix, 0)
	d := time.Since(t)
	switch {
	case d < 48*time.Hour:
		return "today"
	case d < 60*24*time.Hour:
		return strconv.Itoa(int(d.Hours()/24)) + "d"
	case d < 365*24*time.Hour:
		return strconv.Itoa(int(d.Hours()/24/30)) + "mo"
	}
	return t.Format("2006")
}
