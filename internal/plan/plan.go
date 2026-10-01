// Package plan turns a selection into a list of concrete actions and renders
// the exact shell commands they are equivalent to (the dry run).
package plan

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/ivaneidel/mr-plow/internal/catalog"
	"github.com/ivaneidel/mr-plow/internal/safety"
	"github.com/ivaneidel/mr-plow/internal/session"
)

// Kind is what an action does.
type Kind int

const (
	Remove   Kind = iota // rm -rf path
	Trash                // move path to the Trash
	Contents             // rm -rf everything inside path
	Native               // run a tool's own cleanup command
)

// State says whether mr-plow will run an action.
type State int

const (
	Ready     State = iota // will run on Apply
	NeedsRoot              // goes to the sudo script instead
	Blocked                // refused (protected path, missing tool…)
)

// Action is one step of a plan.
type Action struct {
	Item   *session.Item
	Kind   Kind
	State  State
	Reason string // why NeedsRoot/Blocked
	Path   string
	Root   string // scan root the path came from ("" = system extra)
	Size   int64
	Dev    uint64 // identity at scan time (0 = unknown)
	Ino    uint64
	IsDir  bool
	Argv   []string // Native
}

// Plan is the ordered list of actions.
type Plan struct {
	Actions []*Action
	Home    string
	Created time.Time
}

// Context is what Build needs to know about the session.
type Context struct {
	Home, Root string
	Root0      bool // running as root
}

// Build makes a plan from the selected items. Nested selections are merged
// (selecting a folder and something inside it removes the folder once).
func Build(items []*session.Item, ctx Context) *Plan {
	p := &Plan{Home: ctx.Home, Created: time.Now()}
	var flat []*session.Item
	for _, it := range items {
		if len(it.Kids) > 0 {
			flat = append(flat, it.Kids...)
		} else if it.Actionable() {
			flat = append(flat, it)
		}
	}
	// shortest paths first so parents win over their children
	sort.SliceStable(flat, func(i, j int) bool { return len(flat[i].Path) < len(flat[j].Path) })
	var taken []string
	seen := map[string]bool{}
next:
	for _, it := range flat {
		if seen[it.Key] {
			continue
		}
		seen[it.Key] = true
		if it.Mode != catalog.ModeNative {
			for _, t := range taken {
				if it.Path == t || strings.HasPrefix(it.Path, t+"/") {
					continue next
				}
			}
			taken = append(taken, it.Path)
		}
		p.Actions = append(p.Actions, newAction(it, ctx))
	}
	sort.SliceStable(p.Actions, func(i, j int) bool {
		if p.Actions[i].State != p.Actions[j].State {
			return p.Actions[i].State < p.Actions[j].State
		}
		return p.Actions[i].Size > p.Actions[j].Size
	})
	return p
}

func newAction(it *session.Item, ctx Context) *Action {
	a := &Action{Item: it, Path: it.Path, Size: it.Size(), Root: ctx.Root}
	if it.System {
		a.Root = "" // system extras are checked by the denylist only
	}
	switch {
	case it.Node != nil:
		a.Dev, a.Ino, a.IsDir = it.Node.Dev, it.Node.Ino, true
	case it.File != nil:
		a.Dev, a.Ino = it.File.Dev, it.File.Ino
	}
	switch it.Mode {
	case catalog.ModeNative:
		a.Kind = Native
		a.Argv = it.Native
		if _, err := exec.LookPath(it.Native[0]); err != nil {
			a.State, a.Reason = Blocked, "`"+it.Native[0]+"` not installed"
		} else if it.System && !ctx.Root0 {
			a.State, a.Reason = NeedsRoot, "system-wide"
		}
		return a
	case catalog.ModeContents:
		a.Kind = Contents
	default:
		a.Kind = Remove
	}
	if err := CheckPath(a, ctx.Home); err != nil {
		a.State, a.Reason = Blocked, err.Error()
		return a
	}
	if !ctx.Root0 {
		ok := safety.CanRemove(a.Path)
		if a.Kind == Contents {
			ok = safety.CanWriteDir(a.Path)
		}
		if !ok || it.System {
			a.State, a.Reason = NeedsRoot, "no write permission"
		}
	}
	return a
}

// CheckPath runs the denylist for an action. For Contents the folder itself
// is kept, so what must be deletable is "something inside it".
func CheckPath(a *Action, home string) error {
	if a.Kind == Contents {
		return safety.Check(filepath.Join(a.Path, "child"), home, a.Root)
	}
	return safety.Check(a.Path, home, a.Root)
}

// SetTrash switches a Remove action to Trash and back.
func (a *Action) SetTrash(on bool) {
	if on && a.Kind == Remove {
		a.Kind = Trash
	} else if !on && a.Kind == Trash {
		a.Kind = Remove
	}
}

// Q single-quotes a string for POSIX shells.
func Q(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// TrashDir is where Trash actions move things to.
func TrashDir(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, ".Trash")
	}
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "Trash")
	}
	return filepath.Join(home, ".local", "share", "Trash")
}

// Command is the shell command equivalent to the action.
func (a *Action) Command(home string) string {
	var c string
	switch a.Kind {
	case Remove:
		c = "rm -rf -- " + Q(a.Path)
	case Trash:
		dst := TrashDir(home)
		if runtime.GOOS != "darwin" {
			dst = filepath.Join(dst, "files")
		}
		c = "mv -- " + Q(a.Path) + " " + Q(dst+"/")
	case Contents:
		c = "find " + Q(a.Path) + " -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +"
	case Native:
		parts := make([]string, len(a.Argv))
		for i, s := range a.Argv {
			if strings.ContainsAny(s, " '\"$`\\*?") {
				s = Q(s)
			}
			parts[i] = s
		}
		c = strings.Join(parts, " ")
	}
	if a.State == NeedsRoot {
		c = "sudo " + c
	}
	return c
}

// Count returns how many actions are in a state, and their total size.
func (p *Plan) Count(st State) (int, int64) {
	var n int
	var sz int64
	for _, a := range p.Actions {
		if a.State == st {
			n++
			sz += a.Size
		}
	}
	return n, sz
}

// Script renders the plan as a reviewable shell script. With only != nil just
// actions in that state are included. Blocked actions become comments.
func (p *Plan) Script(only *State) string {
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	fmt.Fprintf(&b, "# mr-plow plan · %s\n", p.Created.Format("2006-01-02 15:04"))
	b.WriteString("# Review every line before running it. Nothing here has been run yet.\n")
	b.WriteString("set -u\n\n")
	for _, a := range p.Actions {
		if only != nil && a.State != *only {
			continue
		}
		label := a.Item.Label
		if a.Item.Sub != "" {
			label += " · " + a.Item.Sub
		}
		fmt.Fprintf(&b, "# %s (%s)\n", label, Human(a.Size))
		if a.State == Blocked {
			fmt.Fprintf(&b, "# BLOCKED: %s\n# %s\n\n", a.Reason, a.Command(p.Home))
			continue
		}
		b.WriteString(a.Command(p.Home) + "\n\n")
	}
	return b.String()
}

// Human formats bytes in decimal units, like Finder.
func Human(n int64) string {
	const unit = 1000
	if n < 0 {
		return "-" + Human(-n)
	}
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	v := float64(n) / float64(div)
	if v >= 100 {
		return fmt.Sprintf("%.0f %cB", v, "kMGTPE"[exp])
	}
	return fmt.Sprintf("%.1f %cB", v, "kMGTPE"[exp])
}
