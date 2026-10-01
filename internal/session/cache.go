package session

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ivaneidel/mr-plow/internal/scan"
)

// CacheDir is where the last scan is kept (per real user, even under sudo).
func (s *Session) CacheDir() string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(s.Home, "Library", "Caches", "mr-plow")
	}
	if x := os.Getenv("XDG_CACHE_HOME"); x != "" && !s.ID.Root {
		return filepath.Join(x, "mr-plow")
	}
	return filepath.Join(s.Home, ".cache", "mr-plow")
}

func (s *Session) cacheFile() string { return filepath.Join(s.CacheDir(), "last-scan.gob.gz") }

func (s *Session) loadCache() (*scan.Snapshot, error) {
	f, err := os.Open(s.cacheFile())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sn, err := scan.LoadSnapshot(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}
	if sn.Root.Name != s.Root {
		return nil, os.ErrNotExist
	}
	return sn, nil
}

func (s *Session) saveCache(sn *scan.Snapshot) {
	if err := s.ID.MkdirAll(s.CacheDir()); err != nil {
		return
	}
	tmp := s.cacheFile() + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return
	}
	w := bufio.NewWriter(f)
	err = sn.Save(w)
	if err == nil {
		err = w.Flush()
	}
	f.Close()
	if err != nil {
		os.Remove(tmp)
		return
	}
	s.ID.Chown(tmp)
	_ = os.Rename(tmp, s.cacheFile())
}
