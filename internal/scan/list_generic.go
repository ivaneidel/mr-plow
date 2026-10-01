package scan

import (
	"os"
	"path/filepath"
	"syscall"
)

// GenericLister is the portable baseline: os.ReadDir plus one Lstat per entry
// by full path. It is what most disk tools do, and what --bench compares to.
type GenericLister struct{}

func (GenericLister) Name() string { return "readdir+lstat" }

func (GenericLister) List(path string, dst []Entry) ([]Entry, error) {
	des, err := os.ReadDir(path)
	if err != nil {
		return dst, err
	}
	for _, de := range des {
		fi, err := os.Lstat(filepath.Join(path, de.Name()))
		if err != nil {
			continue
		}
		e := Entry{Name: de.Name(), MTime: fi.ModTime().Unix()}
		switch m := fi.Mode(); {
		case m.IsDir():
			e.Kind = KindDir
		case m.IsRegular():
			e.Kind = KindFile
		case m&os.ModeSymlink != 0:
			e.Kind = KindLink
		}
		if st, ok := fi.Sys().(*syscall.Stat_t); ok {
			e.Alloc = int64(st.Blocks) * 512
			e.Dev = uint64(st.Dev)
			e.Ino = uint64(st.Ino)
			e.Nlink = uint32(st.Nlink)
		} else {
			e.Alloc = fi.Size()
		}
		dst = append(dst, e)
	}
	return dst, nil
}
