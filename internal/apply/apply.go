// Package apply executes a plan, re-checking every path right before touching it.
package apply

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/ivaneidel/mr-plow/internal/disk"
	"github.com/ivaneidel/mr-plow/internal/plan"
	"github.com/ivaneidel/mr-plow/internal/safety"
)

// Options for a run.
type Options struct {
	ID     safety.Identity
	DryRun bool // refuse to do anything (belt and braces for --dry-run)
}

// Result of one action.
type Result struct {
	Action  *plan.Action
	OK      bool
	Skipped string // set when deliberately not run
	Err     error
	Freed   int64 // estimated for rm/trash, measured for native commands
	Output  string
	Took    time.Duration
}

// ErrDryRun is returned when apply is attempted in dry-run mode.
var ErrDryRun = errors.New("dry-run mode: nothing is applied")

// Run executes every Ready action in order. onStep is called before (r == nil)
// and after each action.
func Run(ctx context.Context, p *plan.Plan, o Options, onStep func(i int, r *Result)) ([]Result, error) {
	if o.DryRun {
		return nil, ErrDryRun
	}
	lg := openLog(o.ID)
	defer lg.close()
	var out []Result
	for i, a := range p.Actions {
		if ctx.Err() != nil {
			break
		}
		if onStep != nil {
			onStep(i, nil)
		}
		t0 := time.Now()
		r := one(ctx, a, p.Home)
		r.Took = time.Since(t0)
		lg.write(a, &r, p.Home)
		out = append(out, r)
		if onStep != nil {
			onStep(i, &r)
		}
	}
	return out, nil
}

func one(ctx context.Context, a *plan.Action, home string) Result {
	r := Result{Action: a}
	switch a.State {
	case plan.NeedsRoot:
		r.Skipped = "needs sudo (see script)"
		return r
	case plan.Blocked:
		r.Skipped = a.Reason
		return r
	}
	if a.Kind == plan.Native {
		before, _ := disk.Of(home)
		cmd := exec.CommandContext(ctx, a.Argv[0], a.Argv[1:]...)
		b, err := cmd.CombinedOutput()
		after, _ := disk.Of(home)
		r.Output = tail(string(b), 6)
		r.Err = err
		r.OK = err == nil
		if d := after.Avail - before.Avail; d > 0 {
			r.Freed = d
		}
		return r
	}

	// Re-validate right before deleting: protected paths, and the thing on
	// disk must still be the one we scanned (same inode, same type).
	if err := plan.CheckPath(a, home); err != nil {
		r.Skipped = err.Error()
		return r
	}
	var st unix.Stat_t
	if err := unix.Lstat(a.Path, &st); err != nil {
		if errors.Is(err, unix.ENOENT) {
			r.Skipped = "already gone"
		} else {
			r.Err = err
		}
		return r
	}
	isDir := st.Mode&unix.S_IFMT == unix.S_IFDIR
	if a.Ino != 0 && (uint64(st.Ino) != a.Ino || uint32(st.Dev) != uint32(a.Dev)) {
		r.Skipped = "changed since the scan (different file now)"
		return r
	}
	if a.Ino != 0 && isDir != a.IsDir {
		r.Skipped = "changed since the scan (type differs)"
		return r
	}
	if a.Kind == plan.Contents && !isDir {
		r.Skipped = "not a directory"
		return r
	}

	var err error
	switch a.Kind {
	case plan.Remove:
		err = os.RemoveAll(a.Path) // unlinkat-based; never follows symlinks
	case plan.Contents:
		err = removeContents(a.Path, home, a.Root)
	case plan.Trash:
		err = moveToTrash(a.Path, home)
	}
	r.Err = err
	r.OK = err == nil
	if r.OK {
		r.Freed = a.Size
	}
	return r
}

func removeContents(dir, home, root string) error {
	des, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var first error
	for _, de := range des {
		child := filepath.Join(dir, de.Name())
		if err := safety.Check(child, home, root); err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		if err := os.RemoveAll(child); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func moveToTrash(path, home string) error {
	base := filepath.Base(path)
	if runtime.GOOS == "darwin" {
		dir := plan.TrashDir(home)
		dst := uniqueName(dir, base)
		return rename(path, dst)
	}
	root := plan.TrashDir(home)
	files, info := filepath.Join(root, "files"), filepath.Join(root, "info")
	if err := os.MkdirAll(files, 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(info, 0o700); err != nil {
		return err
	}
	dst := uniqueName(files, base)
	name := filepath.Base(dst)
	ti := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n", path, time.Now().Format("2006-01-02T15:04:05"))
	if err := os.WriteFile(filepath.Join(info, name+".trashinfo"), []byte(ti), 0o600); err != nil {
		return err
	}
	return rename(path, dst)
}

func rename(src, dst string) error {
	err := os.Rename(src, dst)
	if errors.Is(err, unix.EXDEV) {
		return fmt.Errorf("Trash is on another volume; use remove instead")
	}
	return err
}

func uniqueName(dir, base string) string {
	dst := filepath.Join(dir, base)
	if _, err := os.Lstat(dst); err != nil {
		return dst
	}
	return filepath.Join(dir, base+" "+time.Now().Format("15.04.05.000"))
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// LogPath is where every applied action is recorded.
func LogPath(id safety.Identity) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(id.Home, "Library", "Logs", "mr-plow", "actions.log")
	}
	if x := os.Getenv("XDG_STATE_HOME"); x != "" && !id.Root {
		return filepath.Join(x, "mr-plow", "actions.log")
	}
	return filepath.Join(id.Home, ".local", "state", "mr-plow", "actions.log")
}

type logger struct{ f *os.File }

func openLog(id safety.Identity) *logger {
	p := LogPath(id)
	if id.MkdirAll(filepath.Dir(p)) != nil {
		return &logger{}
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return &logger{}
	}
	id.Chown(p)
	return &logger{f: f}
}

func (l *logger) write(a *plan.Action, r *Result, home string) {
	if l.f == nil {
		return
	}
	status := "ok"
	detail := ""
	switch {
	case r.Skipped != "":
		status, detail = "skipped", r.Skipped
	case r.Err != nil:
		status, detail = "error", r.Err.Error()
	}
	fmt.Fprintf(l.f, "%s\t%s\t%s\t%s\t%s\n", time.Now().Format(time.RFC3339), status,
		plan.Human(a.Size), a.Command(home), detail)
}

func (l *logger) close() {
	if l.f != nil {
		l.f.Close()
	}
}
