package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ivaneidel/mr-plow/internal/catalog"
	"github.com/ivaneidel/mr-plow/internal/safety"
)

func mk(t *testing.T, path string, mb int) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, make([]byte, mb<<20), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestItemsOnFakeHome(t *testing.T) {
	home, _ := filepath.EvalSymlinks(t.TempDir())
	os.WriteFile(filepath.Join(home, "p.json"), nil, 0o644)
	mk(t, filepath.Join(home, "Programming", "app", "node_modules", "x.js"), 2)
	os.WriteFile(filepath.Join(home, "Programming", "app", "package.json"), []byte("{}"), 0o644)
	mk(t, filepath.Join(home, ".nvm", "versions", "node", "v1", "lib", "node_modules", "npm.js"), 1)
	mk(t, filepath.Join(home, ".npm", "_cacache", "blob"), 2)
	mk(t, filepath.Join(home, ".cache", "pip", "blob"), 51)
	mk(t, filepath.Join(home, ".cache", "tiny", "blob"), 1)

	s, err := New(Options{Identity: safety.Identity{Home: home}, NoCache: true, NoSystem: true})
	if err != nil {
		t.Fatal(err)
	}
	if !s.IsHome {
		t.Fatal("expected home scan")
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	s.WaitAll()
	v := s.View()

	found := map[string]*Item{}
	for _, g := range v.Groups {
		for _, it := range g.Items {
			found[it.Key] = it
		}
	}
	nm := found["det:node_modules"]
	if nm == nil || len(nm.Kids) != 1 {
		t.Fatalf("want exactly one project node_modules (nvm's is tool-managed), got %+v", nm)
	}
	if nm.Kids[0].Path != filepath.Join(home, "Programming", "app", "node_modules") {
		t.Fatalf("wrong node_modules: %s", nm.Kids[0].Path)
	}
	if found["npm:"+filepath.Join(home, ".npm", "_cacache")] == nil {
		t.Error("npm cache not found")
	}
	if found["x:"+filepath.Join(home, ".cache", "pip")] == nil {
		t.Error("~/.cache/pip (51 MB) not expanded")
	}
	if found["x:"+filepath.Join(home, ".cache", "tiny")] != nil {
		t.Error("~/.cache/tiny (1 MB) should be below the expand threshold")
	}
	if v.Reclaimable <= 0 {
		t.Error("nothing reclaimable")
	}
	for _, it := range found {
		if it.Group == "" {
			t.Errorf("item %s without group", it.Key)
		}
	}
	_ = catalog.Groups
}

func TestToolManaged(t *testing.T) {
	h := "/Users/me"
	cases := map[string]bool{
		h + "/Programming/app/node_modules":                    false,
		h + "/.nvm/versions/node/v1/lib/node_modules":          true,
		h + "/.vscode/extensions/foo/node_modules":             true,
		h + "/Library/Application Support/Code/x/node_modules": true,
		h + "/Programming/py/.venv":                            false,
		"/opt/other/node_modules":                              false,
	}
	for p, want := range cases {
		if got := toolManaged(p, h); got != want {
			t.Errorf("toolManaged(%s) = %v, want %v", p, got, want)
		}
	}
}

func TestHomeThroughSymlink(t *testing.T) {
	base, _ := filepath.EvalSymlinks(t.TempDir())
	real := filepath.Join(base, "real-home")
	mk(t, filepath.Join(real, ".npm", "_cacache", "blob"), 2)
	mk(t, filepath.Join(real, ".nvm", "versions", "node", "v1", "lib", "node_modules", "npm.js"), 1)
	link := filepath.Join(base, "home-link")
	os.Symlink(real, link)
	s, _ := New(Options{Identity: safety.Identity{Home: link}, NoCache: true, NoSystem: true})
	s.Start()
	s.WaitAll()
	var npm, nm bool
	for _, g := range s.View().Groups {
		for _, it := range g.Items {
			npm = npm || it.Label == "npm cache"
			nm = nm || it.Key == "det:node_modules"
		}
	}
	if !npm || nm {
		t.Fatalf("symlinked home: npm cache found=%v (want true), tool node_modules flagged=%v (want false)", npm, nm)
	}
}
