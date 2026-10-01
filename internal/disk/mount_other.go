//go:build !darwin

package disk

import (
	"bufio"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

// mountName finds the longest mount point prefix of path in /proc/self/mounts.
func mountName(_ *unix.Statfs_t, path string) string {
	f, err := os.Open("/proc/self/mounts")
	if err != nil {
		return "/"
	}
	defer f.Close()
	best := "/"
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) < 2 {
			continue
		}
		mp := fs[1]
		if (path == mp || strings.HasPrefix(path, strings.TrimSuffix(mp, "/")+"/")) && len(mp) > len(best) {
			best = mp
		}
	}
	return best
}
