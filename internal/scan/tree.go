package scan

import (
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
)

// Dir flags.
const (
	FlagDenied  uint32 = 1 << iota // could not be listed (EACCES/EPERM)
	FlagMount                      // another filesystem; not descended
	FlagRemoved                    // deleted by an apply; detached from the tree
	FlagSkipped                    // deliberately not descended (e.g. firmlink duplicate)
)

// Dir is one directory in the size tree. Small files are not kept; only their
// totals. Sizes are atomics so the UI can read them while a scan is running.
type Dir struct {
	Name     string // base name; the root holds its full path
	Parent   *Dir
	Dev, Ino uint64
	Tag      string // detector id (e.g. "node_modules"), "" if none
	inTagged bool   // an ancestor has a Tag; detectors skip nested matches

	flags atomic.Uint32
	size  atomic.Int64 // allocated bytes of the whole subtree
	files atomic.Int64 // files in the whole subtree
	kids  atomic.Pointer[[]*Dir]

	selfNewest int64
	newest     atomic.Int64 // newest mtime in subtree; valid once Finalize ran
	done       atomic.Bool  // subtree fully scanned
	pending    atomic.Int32 // children not yet done (+1 for self while listing)
}

func (d *Dir) Size() int64       { return d.size.Load() }
func (d *Dir) Files() int64      { return d.files.Load() }
func (d *Dir) Flags() uint32     { return d.flags.Load() }
func (d *Dir) Has(f uint32) bool { return d.flags.Load()&f != 0 }
func (d *Dir) SetFlag(f uint32)  { d.flags.Or(f) }
func (d *Dir) Newest() int64     { return d.newest.Load() }
func (d *Dir) Done() bool        { return d.done.Load() }

// Children returns the child directories (nil while not yet listed).
func (d *Dir) Children() []*Dir {
	if p := d.kids.Load(); p != nil {
		return *p
	}
	return nil
}

// SortedChildren returns children by size, biggest first.
func (d *Dir) SortedChildren() []*Dir {
	kids := append([]*Dir(nil), d.Children()...)
	sort.Slice(kids, func(i, j int) bool { return kids[i].Size() > kids[j].Size() })
	return kids
}

// Path rebuilds the absolute path by walking up to the root.
func (d *Dir) Path() string {
	var parts []string
	for p := d; p != nil; p = p.Parent {
		parts = append(parts, p.Name)
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return filepath.Join(parts...)
}

// Lookup finds the node for an absolute path under this root, or nil.
func (d *Dir) Lookup(path string) *Dir {
	rel, err := filepath.Rel(d.Name, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
		return nil
	}
	if rel == "." {
		return d
	}
	cur := d
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		var next *Dir
		for _, k := range cur.Children() {
			if k.Name == part {
				next = k
				break
			}
		}
		if next == nil {
			return nil
		}
		cur = next
	}
	return cur
}

// Detach removes d from the tree after it was deleted, subtracting its size
// from every ancestor.
func (d *Dir) Detach() {
	if d.Parent == nil || d.Has(FlagRemoved) {
		return
	}
	sz, n := d.Size(), d.Files()
	for p := d.Parent; p != nil; p = p.Parent {
		p.size.Add(-sz)
		p.files.Add(-n)
	}
	old := d.Parent.Children()
	kids := make([]*Dir, 0, len(old))
	for _, k := range old {
		if k != d {
			kids = append(kids, k)
		}
	}
	d.Parent.kids.Store(&kids)
	d.SetFlag(FlagRemoved)
}

// SubtractFile accounts for a single big file removed from directory d.
func (d *Dir) SubtractFile(size int64) {
	for p := d; p != nil; p = p.Parent {
		p.size.Add(-size)
		p.files.Add(-1)
	}
}

// finalize computes newest-mtime bottom-up. Called once the scan is done.
func (d *Dir) finalize() int64 {
	n := d.selfNewest
	for _, k := range d.Children() {
		if v := k.finalize(); v > n {
			n = v
		}
	}
	d.newest.Store(n)
	return n
}

// Walk visits every node depth-first.
func (d *Dir) Walk(fn func(*Dir) bool) {
	if !fn(d) {
		return
	}
	for _, k := range d.Children() {
		k.Walk(fn)
	}
}

// File is a big file remembered by the scanner.
type File struct {
	Path     string
	Size     int64
	MTime    int64
	Dev, Ino uint64
}
