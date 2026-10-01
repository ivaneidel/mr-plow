//go:build darwin || linux

package scan

import (
	"os"

	"golang.org/x/sys/unix"
)

// FstatatLister opens each directory once and stats entries relative to the
// open descriptor, so the kernel never re-resolves the full path.
type FstatatLister struct{}

func (FstatatLister) Name() string { return "readdir+fstatat" }

func (FstatatLister) List(path string, dst []Entry) ([]Entry, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return dst, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	names, err := f.Readdirnames(-1)
	if err != nil && len(names) == 0 {
		return dst, err
	}
	var st unix.Stat_t
	for _, name := range names {
		if unix.Fstatat(fd, name, &st, unix.AT_SYMLINK_NOFOLLOW) != nil {
			continue
		}
		e := Entry{
			Name:  name,
			Alloc: int64(st.Blocks) * 512,
			Dev:   uint64(st.Dev),
			Ino:   uint64(st.Ino),
			Nlink: uint32(st.Nlink),
			MTime: int64(st.Mtim.Sec),
		}
		switch uint32(st.Mode) & unix.S_IFMT {
		case unix.S_IFDIR:
			e.Kind = KindDir
		case unix.S_IFREG:
			e.Kind = KindFile
		case unix.S_IFLNK:
			e.Kind = KindLink
		}
		dst = append(dst, e)
	}
	return dst, nil
}
