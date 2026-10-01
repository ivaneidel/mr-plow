package safety

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheck(t *testing.T) {
	home := "/Users/me"
	root := "/Users/me"
	deny := []string{
		"/", "/System", "/System/Library/Foo", "/usr/local/bin", "/Applications", "/Users",
		home, home + "/Documents", home + "/Library", home + "/Programming", home + "/.ssh",
		home + "/.config", home + "/.Trash", "/Users/other/thing", "relative/path",
		home + "/Programming/../.ssh/id_rsa", "/etc/hosts",
	}
	for _, p := range deny {
		if Check(p, home, root) == nil {
			t.Errorf("Check(%q) allowed, want denied", p)
		}
	}
	allow := []string{
		home + "/Programming/proj/node_modules", home + "/Library/Caches/com.foo",
		home + "/.npm/_cacache", home + "/.Trash/old.dmg", home + "/Downloads/ubuntu.iso",
	}
	for _, p := range allow {
		if err := Check(p, home, root); err != nil {
			t.Errorf("Check(%q) = %v, want allowed", p, err)
		}
	}
}

func TestCheckOutsideRoot(t *testing.T) {
	if Check("/Users/me/Music/x", "/Users/me", "/Users/me/Programming") == nil {
		t.Fatal("path outside scan root was allowed")
	}
}

func TestMountPointDenied(t *testing.T) {
	// /dev is a separate filesystem on both macOS and Linux
	if err := Check(filepath.Clean("/dev"), "", ""); err == nil {
		t.Fatal("/dev allowed")
	}
}

func TestHomeBehindSymlink(t *testing.T) {
	base, _ := filepath.EvalSymlinks(t.TempDir())
	real := filepath.Join(base, "real")
	if err := os.MkdirAll(filepath.Join(real, "Documents"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	os.Symlink(real, link)
	for _, p := range []string{real, filepath.Join(real, "Documents")} {
		if Check(p, link, "") == nil {
			t.Errorf("Check(%s) with home=%s allowed", p, link)
		}
	}
	if err := Check(filepath.Join(real, "Documents", "x"), link, ""); err != nil {
		t.Errorf("inside Documents should be allowed: %v", err)
	}
}
