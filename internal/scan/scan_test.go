package scan

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func scanWith(t *testing.T, root string, l Lister, detect func([]Entry, string) string) *Scanner {
	t.Helper()
	s, err := New(Options{Root: root, Lister: l, Detect: detect, BigMin: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	s.Wait()
	return s
}

func TestScanFixture(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	outside := filepath.Join(base, "outside")
	write(t, filepath.Join(root, "a", "one.bin"), 3<<20)
	write(t, filepath.Join(root, "a", "b", "two.bin"), 2<<20)
	write(t, filepath.Join(root, "proj", "package.json"), 10)
	write(t, filepath.Join(root, "proj", "node_modules", "x", "index.js"), 4096)
	write(t, filepath.Join(root, "proj", "node_modules", "y", "node_modules", "z.js"), 4096)
	write(t, filepath.Join(outside, "huge.bin"), 8<<20)
	// symlink to a big directory outside the root must not be followed
	if err := os.Symlink(outside, filepath.Join(root, "link-out")); err != nil {
		t.Fatal(err)
	}
	// hard link: counted once
	if err := os.Link(filepath.Join(root, "a", "one.bin"), filepath.Join(root, "one-again.bin")); err != nil {
		t.Fatal(err)
	}

	detect := func(sib []Entry, name string) string {
		if name == "node_modules" {
			return "node_modules"
		}
		return ""
	}
	var sizes []int64
	for _, l := range Listers() {
		s := scanWith(t, root, l, detect)
		r := s.Root()
		if !r.Done() {
			t.Errorf("%s: root not done", l.Name())
		}
		if r.Size() >= 8<<20 {
			t.Errorf("%s: symlink was followed (size %d)", l.Name(), r.Size())
		}
		if r.Size() < 5<<20 || r.Size() > 6<<20 {
			t.Errorf("%s: size %d, want ~5 MiB (hard link counted once)", l.Name(), r.Size())
		}
		sizes = append(sizes, r.Size())
		tagged := s.Tagged()
		if len(tagged) != 1 || tagged[0].Path() != filepath.Join(r.Name, "proj", "node_modules") {
			t.Errorf("%s: tagged = %v, want only the outer node_modules", l.Name(), tagged)
		}
		big := s.BigFiles()
		if len(big) < 2 || big[0].Size < big[1].Size {
			t.Errorf("%s: big files %v", l.Name(), big)
		}
		if d := r.Lookup(filepath.Join(r.Name, "a", "b")); d == nil || d.Files() != 1 {
			t.Errorf("%s: lookup a/b failed", l.Name())
		}
	}
	for i := range sizes {
		if sizes[i] != sizes[0] {
			t.Errorf("listers disagree: %v", sizes)
		}
	}
}

func TestDetachUpdatesAncestors(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "x", "y", "f.bin"), 1<<20)
	write(t, filepath.Join(root, "keep.bin"), 1<<20)
	s := scanWith(t, root, nil, nil)
	r := s.Root()
	before := r.Size()
	y := r.Lookup(filepath.Join(r.Name, "x", "y"))
	sz := y.Size()
	y.Detach()
	if r.Size() != before-sz {
		t.Fatalf("root size %d, want %d", r.Size(), before-sz)
	}
	if r.Lookup(filepath.Join(r.Name, "x", "y")) != nil {
		t.Fatal("detached node still reachable")
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "p", "node_modules", "a.js"), 5000)
	write(t, filepath.Join(root, "big.iso"), 2<<20)
	s := scanWith(t, root, nil, func(_ []Entry, n string) string {
		if n == "node_modules" {
			return "nm"
		}
		return ""
	})
	sn := s.Snapshot()
	f, _ := os.CreateTemp(t.TempDir(), "snap")
	if err := sn.Save(f); err != nil {
		t.Fatal(err)
	}
	f.Seek(0, 0)
	got, err := LoadSnapshot(f)
	if err != nil {
		t.Fatal(err)
	}
	if got.Root.Size() != sn.Root.Size() || len(got.Tagged) != 1 || len(got.Big) != 1 || !got.Cached {
		t.Fatalf("round trip mismatch: %d/%d tagged=%d big=%d", got.Root.Size(), sn.Root.Size(), len(got.Tagged), len(got.Big))
	}
	if got.Root.Lookup(filepath.Join(got.Root.Name, "p", "node_modules")) == nil {
		t.Fatal("lookup after load failed")
	}
}
