package apply

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ivaneidel/mr-plow/internal/catalog"
	"github.com/ivaneidel/mr-plow/internal/plan"
	"github.com/ivaneidel/mr-plow/internal/safety"
	"github.com/ivaneidel/mr-plow/internal/scan"
	"github.com/ivaneidel/mr-plow/internal/session"
)

type fixture struct {
	home, root, outside string
	snap                *scan.Snapshot
	id                  safety.Identity
}

func write(t *testing.T, path string, size int) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newFixture(t *testing.T) *fixture {
	base, _ := filepath.EvalSymlinks(t.TempDir())
	f := &fixture{home: filepath.Join(base, "home"), outside: filepath.Join(base, "outside")}
	f.root = filepath.Join(f.home, "Programming")
	write(t, filepath.Join(f.root, "app", "package.json"), 10)
	write(t, filepath.Join(f.root, "app", "node_modules", "lib", "a.js"), 50000)
	write(t, filepath.Join(f.root, "app", "src", "main.js"), 100)
	write(t, filepath.Join(f.root, "old", "ubuntu.iso"), 3<<20)
	write(t, filepath.Join(f.outside, "precious.txt"), 100)
	// a symlink inside the doomed folder pointing at precious data outside
	os.Symlink(f.outside, filepath.Join(f.root, "app", "node_modules", "evil-link"))
	write(t, filepath.Join(f.home, ".Trash", "junk.bin"), 1000)
	os.MkdirAll(filepath.Join(f.home, "Library", "Logs"), 0o755)
	f.id = safety.Identity{Home: f.home, UID: os.Getuid(), GID: os.Getgid()}
	s, err := scan.New(scan.Options{Root: f.root, BigMin: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	s.Wait()
	f.snap = s.Snapshot()
	return f
}

func (f *fixture) dirItem(t *testing.T, rel string, mode catalog.Mode) *session.Item {
	p := filepath.Join(f.root, rel)
	d := f.snap.Root.Lookup(p)
	if d == nil {
		t.Fatalf("no node for %s", p)
	}
	return &session.Item{Key: p, Label: rel, Path: p, Node: d, Mode: mode}
}

func (f *fixture) ctx() plan.Context { return plan.Context{Home: f.home, Root: f.root} }

func exists(p string) bool { _, err := os.Lstat(p); return err == nil }

func TestRemoveDoesNotFollowSymlinks(t *testing.T) {
	f := newFixture(t)
	p := plan.Build([]*session.Item{f.dirItem(t, "app/node_modules", catalog.ModeRemove)}, f.ctx())
	if len(p.Actions) != 1 || p.Actions[0].State != plan.Ready {
		t.Fatalf("plan: %+v", p.Actions[0])
	}
	res, err := Run(context.Background(), p, Options{ID: f.id}, nil)
	if err != nil || !res[0].OK {
		t.Fatalf("run: %v %+v", err, res)
	}
	if exists(filepath.Join(f.root, "app", "node_modules")) {
		t.Fatal("node_modules still there")
	}
	if !exists(filepath.Join(f.outside, "precious.txt")) {
		t.Fatal("followed the symlink and deleted data outside!")
	}
	if !exists(filepath.Join(f.root, "app", "src", "main.js")) {
		t.Fatal("deleted a sibling")
	}
	if !exists(LogPath(f.id)) {
		t.Fatal("no action log written")
	}
}

func TestDryRunTouchesNothing(t *testing.T) {
	f := newFixture(t)
	p := plan.Build([]*session.Item{f.dirItem(t, "app", catalog.ModeRemove)}, f.ctx())
	_, err := Run(context.Background(), p, Options{ID: f.id, DryRun: true}, nil)
	if !errors.Is(err, ErrDryRun) {
		t.Fatalf("err = %v", err)
	}
	if !exists(filepath.Join(f.root, "app", "node_modules", "lib", "a.js")) {
		t.Fatal("dry run deleted something")
	}
}

func TestSwappedPathIsSkipped(t *testing.T) {
	f := newFixture(t)
	p := plan.Build([]*session.Item{f.dirItem(t, "app/node_modules", catalog.ModeRemove)}, f.ctx())
	nm := filepath.Join(f.root, "app", "node_modules")
	// between scan and apply, the folder is replaced by a symlink to precious data
	os.RemoveAll(nm)
	os.Symlink(f.outside, nm)
	res, _ := Run(context.Background(), p, Options{ID: f.id}, nil)
	if res[0].Skipped == "" {
		t.Fatalf("swapped path not skipped: %+v", res[0])
	}
	if !exists(filepath.Join(f.outside, "precious.txt")) || !exists(nm) {
		t.Fatal("swapped path was acted on")
	}
}

func TestNestedSelectionMerged(t *testing.T) {
	f := newFixture(t)
	items := []*session.Item{f.dirItem(t, "app/node_modules/lib", catalog.ModeRemove), f.dirItem(t, "app", catalog.ModeRemove)}
	p := plan.Build(items, f.ctx())
	if len(p.Actions) != 1 || p.Actions[0].Path != filepath.Join(f.root, "app") {
		t.Fatalf("want one action for app, got %d", len(p.Actions))
	}
}

func TestBlockedPaths(t *testing.T) {
	f := newFixture(t)
	rootItem := &session.Item{Key: "r", Path: f.root, Node: f.snap.Root, Mode: catalog.ModeRemove}
	homeItem := &session.Item{Key: "h", Path: f.home, Mode: catalog.ModeRemove}
	outItem := &session.Item{Key: "o", Path: filepath.Join(f.outside, "precious.txt"), Mode: catalog.ModeRemove}
	p := plan.Build([]*session.Item{rootItem, homeItem, outItem}, f.ctx())
	for _, a := range p.Actions {
		if a.State != plan.Blocked {
			t.Errorf("%s: state %v, want Blocked", a.Path, a.State)
		}
	}
	res, _ := Run(context.Background(), p, Options{ID: f.id}, nil)
	for _, r := range res {
		if r.OK {
			t.Errorf("blocked action ran: %s", r.Action.Path)
		}
	}
	if !exists(f.root) || !exists(filepath.Join(f.outside, "precious.txt")) {
		t.Fatal("protected path deleted")
	}
}

func TestTrashAndContents(t *testing.T) {
	f := newFixture(t)
	// file item → Trash
	iso := filepath.Join(f.root, "old", "ubuntu.iso")
	var file *scan.File
	for i := range f.snap.Big {
		if f.snap.Big[i].Path == iso {
			file = &f.snap.Big[i]
		}
	}
	if file == nil {
		t.Fatal("iso not in big files")
	}
	p := plan.Build([]*session.Item{{Key: iso, Path: iso, File: file, Mode: catalog.ModeRemove}}, f.ctx())
	p.Actions[0].SetTrash(true)
	res, _ := Run(context.Background(), p, Options{ID: f.id}, nil)
	if !res[0].OK || exists(iso) {
		t.Fatalf("trash failed: %+v", res[0])
	}
	trashed := filepath.Join(plan.TrashDir(f.home), "ubuntu.iso")
	if _, err := os.Stat(trashed); err != nil {
		if _, err2 := os.Stat(filepath.Join(plan.TrashDir(f.home), "files", "ubuntu.iso")); err2 != nil {
			t.Fatalf("not found in trash: %v", err)
		}
	}

	// Contents: empty the fake ~/.Trash, keep the folder
	trash := filepath.Join(f.home, ".Trash")
	pc := plan.Build([]*session.Item{{Key: "t", Path: trash, Mode: catalog.ModeContents}}, plan.Context{Home: f.home, Root: f.home})
	if pc.Actions[0].State != plan.Ready {
		t.Fatalf("contents action not ready: %s", pc.Actions[0].Reason)
	}
	res, _ = Run(context.Background(), pc, Options{ID: f.id}, nil)
	if !res[0].OK {
		t.Fatalf("contents: %+v", res[0])
	}
	if !exists(trash) || exists(filepath.Join(trash, "junk.bin")) {
		t.Fatal("contents mode wrong")
	}
}

func TestScriptQuoting(t *testing.T) {
	a := &plan.Action{Kind: plan.Remove, Path: "/tmp/it's here", Item: &session.Item{Label: "x"}}
	if got := a.Command(""); got != `rm -rf -- '/tmp/it'\''s here'` {
		t.Fatalf("got %s", got)
	}
}
