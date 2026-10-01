package scan

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Bench scans root once per lister (after one warm-up pass) and compares
// with `du -sk`. Results are printed to w.
func Bench(w io.Writer, root string, runs int) error {
	if runs < 1 {
		runs = 1
	}
	fmt.Fprintf(w, "mr-plow bench · root %s · %d run(s) each, warm cache\n\n", root, runs)
	warm, err := New(Options{Root: root})
	if err != nil {
		return err
	}
	warm.Start()
	warm.Wait()
	fmt.Fprintf(w, "%-18s %10s %10s %12s %12s %10s\n", "method", "time", "dirs", "files", "files/s", "size")
	for _, l := range Listers() {
		var best time.Duration
		var p Progress
		var size int64
		for i := 0; i < runs; i++ {
			s, err := New(Options{Root: root, Lister: l})
			if err != nil {
				return err
			}
			s.Start()
			s.Wait()
			p = s.Progress()
			size = s.Root().Size()
			if best == 0 || p.Elapsed < best {
				best = p.Elapsed
			}
		}
		fmt.Fprintf(w, "%-18s %10s %10d %12d %12.0f %10s\n", l.Name(), best.Round(time.Millisecond),
			p.Dirs, p.Files, float64(p.Files)/best.Seconds(), human(size))
	}
	t0 := time.Now()
	out, _ := exec.Command("du", "-sk", root).Output() // du exits non-zero on unreadable dirs; output is still valid
	dt := time.Since(t0)
	if f := strings.Fields(string(out)); len(f) > 0 {
		kb, _ := strconv.ParseInt(f[0], 10, 64)
		fmt.Fprintf(w, "%-18s %10s %10s %12s %12s %10s\n", "du -sk", dt.Round(time.Millisecond), "", "", "", human(kb*1024))
	}
	return nil
}

func human(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
