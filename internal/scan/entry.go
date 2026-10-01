// Package scan walks a directory tree in parallel and keeps a compact,
// directory-only size tree plus a heap of the biggest files.
package scan

// Kind is the type of a directory entry, as reported without following links.
type Kind uint8

const (
	KindOther Kind = iota
	KindFile
	KindDir
	KindLink
)

// Entry is one directory entry with just the attributes the scanner needs.
type Entry struct {
	Name  string
	Kind  Kind
	Alloc int64 // allocated bytes on disk (what deleting would free)
	Dev   uint64
	Ino   uint64
	Nlink uint32
	MTime int64 // unix seconds
	Mount bool  // entry is a mount point (when the lister can tell)
}

// Lister reads one directory. Implementations append to dst and return it so
// buffers can be reused across calls.
type Lister interface {
	Name() string
	List(path string, dst []Entry) ([]Entry, error)
}

// Listers returns every lister usable on this platform, fastest first.
func Listers() []Lister {
	return append(platformListers(), GenericLister{})
}

// DefaultLister is the fastest lister available on this platform.
func DefaultLister() Lister { return Listers()[0] }

// ListerByName finds a lister by its Name(), or nil.
func ListerByName(name string) Lister {
	for _, l := range Listers() {
		if l.Name() == name {
			return l
		}
	}
	return nil
}
