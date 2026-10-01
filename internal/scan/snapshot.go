package scan

import (
	"compress/gzip"
	"encoding/gob"
	"io"
	"time"
)

// Snapshot is a finished (or loaded-from-cache) scan: what the UI renders.
type Snapshot struct {
	Root     *Dir
	Big      []File
	Tagged   []*Dir
	Denied   []string
	Progress Progress
	Taken    time.Time
	Lister   string
	Cached   bool // loaded from disk, not scanned this session
}

// Snapshot returns a view of the scan; safe to call while it runs.
func (s *Scanner) Snapshot() *Snapshot {
	return &Snapshot{
		Root: s.root, Big: s.BigFiles(), Tagged: s.Tagged(), Denied: s.DeniedPaths(),
		Progress: s.Progress(), Taken: s.start, Lister: s.list.Name(),
	}
}

const snapshotVersion = 1

type wireNode struct {
	Name           string
	Parent         int32
	Dev, Ino       uint64
	Tag            string
	Flags          uint32
	Size, Files    int64
	Newest, SelfNw int64
}

type wireSnap struct {
	Version  int
	Nodes    []wireNode
	Big      []File
	Denied   []string
	Progress Progress
	Taken    time.Time
	Lister   string
}

// Save writes a finished snapshot (gob + gzip).
func (sn *Snapshot) Save(w io.Writer) error {
	ws := wireSnap{Version: snapshotVersion, Big: sn.Big, Denied: sn.Denied, Progress: sn.Progress, Taken: sn.Taken, Lister: sn.Lister}
	idx := map[*Dir]int32{}
	sn.Root.Walk(func(d *Dir) bool {
		parent := int32(-1)
		if d.Parent != nil {
			parent = idx[d.Parent]
		}
		idx[d] = int32(len(ws.Nodes))
		ws.Nodes = append(ws.Nodes, wireNode{
			Name: d.Name, Parent: parent, Dev: d.Dev, Ino: d.Ino, Tag: d.Tag, Flags: d.Flags(),
			Size: d.Size(), Files: d.Files(), Newest: d.Newest(), SelfNw: d.selfNewest,
		})
		return true
	})
	zw, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
	if err := gob.NewEncoder(zw).Encode(&ws); err != nil {
		return err
	}
	return zw.Close()
}

// LoadSnapshot reads a snapshot written by Save.
func LoadSnapshot(r io.Reader) (*Snapshot, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	var ws wireSnap
	if err := gob.NewDecoder(zr).Decode(&ws); err != nil {
		return nil, err
	}
	if ws.Version != snapshotVersion || len(ws.Nodes) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	nodes := make([]*Dir, len(ws.Nodes))
	kids := make([][]*Dir, len(ws.Nodes))
	sn := &Snapshot{Big: ws.Big, Denied: ws.Denied, Progress: ws.Progress, Taken: ws.Taken, Lister: ws.Lister, Cached: true}
	for i, w := range ws.Nodes {
		d := &Dir{Name: w.Name, Dev: w.Dev, Ino: w.Ino, Tag: w.Tag, selfNewest: w.SelfNw}
		d.flags.Store(w.Flags)
		d.size.Store(w.Size)
		d.files.Store(w.Files)
		d.newest.Store(w.Newest)
		d.done.Store(true)
		if w.Parent >= 0 {
			d.Parent = nodes[w.Parent]
			d.inTagged = d.Parent.inTagged || d.Parent.Tag != ""
			kids[w.Parent] = append(kids[w.Parent], d)
		}
		nodes[i] = d
		if d.Tag != "" {
			sn.Tagged = append(sn.Tagged, d)
		}
	}
	for i := range nodes {
		k := kids[i]
		nodes[i].kids.Store(&k)
	}
	sn.Root = nodes[0]
	return sn, nil
}
