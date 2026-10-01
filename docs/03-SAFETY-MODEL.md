# 03 · Safety model

mr-plow deletes files, so these are the rules, each with a test or check behind it.

## Things it never does

| rule | where | verified by |
|---|---|---|
| No network code | `make nonet-check` fails if `net`, `net/http`, `crypto/tls`… are linked | Makefile, part of `make test` |
| Never follows symlinks while scanning | listers use `lstat`/`O_NOFOLLOW` semantics; a link counts as a link | `scan.TestScanFixture` (8 MiB behind a link not counted) |
| Never follows symlinks while deleting | `os.RemoveAll` uses `unlinkat` and never descends through a link | `apply.TestRemoveDoesNotFollowSymlinks` |
| Never deletes `/`, `$HOME`, anything above home, home's top-level folders (`~/Documents`, `~/Library`, `~/Projects`…) or key dot-dirs (`~/.ssh`, `~/.config`, `~/.gnupg`, `~/.aws`…) | `safety.Check` | `safety.TestCheck` |
| Never deletes system paths (`/System`, `/usr`, `/bin`, `/sbin`, `/etc`, `/private/var/db`, `/Library/Apple`…) or `/Applications` itself | `safety.Check` | `safety.TestCheck` |
| Never deletes a mount point | dev of the path ≠ dev of its parent | `safety.TestMountPointDenied` |
| Never deletes outside the scan root | `safety.Check(path, home, root)` | `safety.TestCheckOutsideRoot`, `apply.TestBlockedPaths` |
| Never deletes something that changed since the scan | right before acting, `lstat` again: inode, device and type must match the scan | `apply.TestSwappedPathIsSkipped` (folder swapped for a symlink to precious data → skipped) |
| Never elevates itself | no `sudo` call anywhere; root-only actions go into a script you run yourself | code review: `sudo` is never executed; it only appears in display text and in lines of exported scripts |
| Never runs anything without a dry run first | Apply is only reachable from the dry-run screen, after a `y` confirmation (or typing `yes` as root) | UI flow |

The denylist runs twice: when the plan is built (the action shows as **Blocked** with the reason) and again right before the deletion.

## Dry run

- The dry-run screen shows each action's exact shell equivalent, wrapped and never truncated: `rm -rf -- '<path>'`, `mv -- '<path>' '~/.Trash/'`, `find '<dir>' -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +` for "empty this folder", or the tool's own command (`docker system prune -a -f`, `brew cleanup -s --prune=all`, `go clean -modcache`, `xcrun simctl delete unavailable`, `pnpm store prune`…).
- `e` exports the plan as a reviewed shell script (blocked items appear as comments).
- `--dry-run` or `MR_PLOW_DRY_RUN=1` disables Apply entirely. `apply.Run` refuses as well (`apply.TestDryRunTouchesNothing`).

## Native commands instead of rm

Some things shouldn't be `rm`'d:
- Docker's VM disk: removing `Docker.raw` loses every image and volume.
- The Go module cache: its files are read-only, so `rm -rf` fails half-way.
- The pnpm store: projects hard-link into it.
- Homebrew: `cleanup` also drops outdated installs.

For these the action is the tool's own command, shown verbatim. If the tool isn't installed, the action is Blocked.

## sudo

- **Normal use needs no sudo.** Everything under your home is yours.
- **Without sudo:**
  - Items mr-plow can't remove (no write+search permission on the parent) are marked 🔒 **needs sudo**.
  - System-wide items are marked the same way.
  - Neither is run; after Apply they're written to `mr-plow-plan-sudo.sh` with a hint to read it and then run `sudo sh <file>`.
- **With sudo** (`sudo mr-plow`):
  - `SUDO_USER` identifies the real user, and their home is scanned, not `/var/root`.
  - The cache, action log and exported scripts are written into their home and chowned back to them.
  - A red **ROOT** badge is shown.
  - Apply requires typing `yes`.
- **sudo doesn't get past macOS privacy protection.** Folders like `~/Library/Mail`, Messages and Safari need Full Disk Access granted to your terminal app. mr-plow counts unreadable folders in the header (`⚠ N unreadable`) and explains where to grant access (System Settings › Privacy & Security › Full Disk Access).

## Action log

Every applied action is appended to `~/Library/Logs/mr-plow/actions.log` on macOS, or `$XDG_STATE_HOME/mr-plow/actions.log` (default `~/.local/state/…`) on Linux. Each line has the timestamp, status (ok / skipped / error), size, exact command and detail.

## Known limits

- Sizes are allocated bytes. APFS clones (copy-on-write copies) share blocks that a scanner can't see, so freeing a cloned file may free less than shown. Hard links are counted once (first seen wins). For a pnpm-linked `node_modules`, the bytes mostly belong to the store.
- "Freed" in the report is the scan-time estimate for `rm`/Trash and the measured change in free space for native commands. The header gauge shows the real before → after.
- The sudo path (`safety.Identify`, chown-back of cache/log/scripts, the `yes` confirmation) has **no automated tests** and wasn't run under a real `sudo` during development. It is the least-verified part.
