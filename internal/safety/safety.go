// Package safety decides what mr-plow is allowed to touch and who it runs as.
package safety

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// Identity is who we act for.
type Identity struct {
	Root     bool   // effective uid 0
	SudoUser string // the human behind sudo, if any
	Home     string // that human's home (not root's)
	UID, GID int    // owner for files we write (cache, logs)
}

// Identify works out the real user even under sudo.
func Identify() Identity {
	id := Identity{Root: os.Geteuid() == 0, UID: os.Getuid(), GID: os.Getgid()}
	id.Home, _ = os.UserHomeDir()
	if id.Root {
		if su := os.Getenv("SUDO_USER"); su != "" && su != "root" {
			id.SudoUser = su
			if u, err := user.Lookup(su); err == nil {
				id.Home = u.HomeDir
			} else if runtime.GOOS == "darwin" {
				id.Home = "/Users/" + su
			} else {
				id.Home = "/home/" + su
			}
			id.UID, _ = strconv.Atoi(os.Getenv("SUDO_UID"))
			id.GID, _ = strconv.Atoi(os.Getenv("SUDO_GID"))
		}
	}
	return id
}

// Chown hands a file we created back to the real user when running under sudo.
func (id Identity) Chown(path string) {
	if id.Root && id.SudoUser != "" {
		_ = os.Lchown(path, id.UID, id.GID)
	}
}

// MkdirAll creates a directory owned by the real user.
func (id Identity) MkdirAll(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if id.Root && id.SudoUser != "" {
		for p := dir; strings.HasPrefix(p, id.Home+"/"); p = filepath.Dir(p) {
			id.Chown(p)
		}
	}
	return nil
}

// ErrDenied is returned for paths mr-plow will never delete.
var ErrDenied = errors.New("protected path")

var systemRoots = []string{
	"/System", "/usr", "/bin", "/sbin", "/etc", "/private/etc", "/private/var/db",
	"/dev", "/proc", "/sys", "/boot", "/lib", "/lib64", "/var/lib/dpkg", "/var/lib/rpm",
	"/Library/Apple", "/Library/Extensions", "/Library/Frameworks",
}

// exact paths that are never deleted themselves (their contents may be).
var exactNever = []string{
	"/", "/Applications", "/Users", "/home", "/Library", "/Volumes", "/var", "/private",
	"/private/var", "/tmp", "/private/tmp", "/opt", "/root", "/mnt", "/media", "/srv",
}

// Check returns an error if path must never be deleted. root is the scan root
// the path came from ("" to skip that check).
func Check(path, home, root string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%w: not absolute: %s", ErrDenied, path)
	}
	clean := filepath.Clean(path)
	if clean != path && clean+"/" != path {
		return fmt.Errorf("%w: not clean: %s", ErrDenied, path)
	}
	for _, p := range exactNever {
		if clean == p {
			return fmt.Errorf("%w: %s", ErrDenied, clean)
		}
	}
	for _, p := range systemRoots {
		if clean == p || strings.HasPrefix(clean, p+"/") {
			return fmt.Errorf("%w: system location %s", ErrDenied, p)
		}
	}
	if home != "" {
		h := filepath.Clean(home)
		if r, err := filepath.EvalSymlinks(h); err == nil && r != h && strings.HasPrefix(clean, r) {
			h = r // path came from a symlink-free scan; compare like with like
		}
		if clean == h || strings.HasPrefix(h, clean+"/") {
			return fmt.Errorf("%w: home or above", ErrDenied)
		}
		// top-level folders of home (~/Documents, ~/Library, ~/Programming…) themselves
		if filepath.Dir(clean) == h {
			base := filepath.Base(clean)
			if base == ".Trash" || !strings.HasPrefix(base, ".") || isKeyDotDir(base) {
				return fmt.Errorf("%w: top-level home folder %s", ErrDenied, base)
			}
		}
	}
	if root != "" {
		r := filepath.Clean(root)
		if !strings.HasPrefix(clean, r+"/") {
			return fmt.Errorf("%w: outside scan root %s", ErrDenied, r)
		}
	}
	if isMountPoint(clean) {
		return fmt.Errorf("%w: mount point", ErrDenied)
	}
	return nil
}

// dot-dirs at the top of home holding config or keys, never removed whole.
func isKeyDotDir(base string) bool {
	switch base {
	case ".ssh", ".gnupg", ".config", ".local", ".aws", ".kube", ".docker", ".zshrc", ".bashrc", ".gitconfig", ".claude":
		return true
	}
	return false
}

func isMountPoint(path string) bool {
	var a, b unix.Stat_t
	if unix.Lstat(path, &a) != nil || unix.Lstat(filepath.Dir(path), &b) != nil {
		return false
	}
	return a.Mode&unix.S_IFMT == unix.S_IFDIR && a.Dev != b.Dev
}

// CanRemove reports whether the current user can unlink path (write+search on
// its parent). Deeper permission problems only show up when deleting.
func CanRemove(path string) bool {
	return unix.Access(filepath.Dir(path), unix.W_OK|unix.X_OK) == nil
}

// CanWriteDir reports whether entries inside dir can be removed.
func CanWriteDir(dir string) bool {
	return unix.Access(dir, unix.W_OK|unix.X_OK) == nil
}
