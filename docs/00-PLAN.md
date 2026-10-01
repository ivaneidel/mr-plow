# mr-plow: plan

## Context
Disk space hides in places nobody remembers: caches, containers, old node_modules, ISOs, someone's music collection. The goal is a terminal tool that is tiny, fast and safe. It finds big or garbage directories, explains what each one is, lets you pick some, shows an exact dry run (`rm -rf …` / native command per item), then applies it. No third-party cleaner gets to delete your files.

**Decisions so far:** Go + Bubble Tea · macOS first, Linux too · permanent `rm` by default with an optional per-item Trash toggle, plus "export as script" · name/command: **`mr-plow`** · English + Spanish UI, colours adapt to a light/dark terminal · docs in `docs/` with a `STATUS.md` updated after every milestone.

## Stack
- Go 1.25+. A `.tool-versions` file (`golang 1.25.3`) is included for asdf users.
- `charmbracelet/bubbletea`, `bubbles`, `lipgloss` for the UI. `golang.org/x/sys/unix` for syscalls. Nothing else unless it earns its place.
- Single binary:
  - macOS: built with cgo, but it only links `libSystem`, which every macOS binary links anyway. Built for arm64 and amd64 and merged into one universal binary with `lipo`.
  - Linux: `CGO_ENABLED=0`, fully static.

## Layout
```
mr-plow/
  cmd/mr-plow/main.go         flags: --bench, --root, --no-cache, --lang, --script-out
  internal/scan/              parallel walker, tree model, hardlink dedupe
    walk.go                   work-stealing pool, streams progress events
    readdir_darwin.go         getattrlistbulk(2) via cgo (fast path)
    readdir_linux.go          getdents64 + fstatat(dirfd) via x/sys/unix
    readdir_generic.go        os.ReadDir + Lstat fallback (also the bench baseline)
  internal/catalog/           known garbage locations + pattern detectors (node_modules, target, .venv, DerivedData, Pods…)
    darwin.go / linux.go      platform catalogs: path, label, explanation, safety level, preferred native command
  internal/plan/              selection → Plan (list of Actions: Remove | Trash | NativeCmd | NeedsSudo)
  internal/apply/             executor: re-checks each path, denylist, logging, progress
  internal/safety/            denylist, root/SUDO_USER detection, Full Disk Access / EPERM accounting
  internal/cache/             snapshot save/load (gob, ~/Library/Caches/mr-plow or $XDG_CACHE_HOME)
  internal/disk/              volume usage (statfs), APFS purgeable/snapshot hints
  internal/i18n/              en/es string tables
  internal/ui/                Bubble Tea models: overview, suspects, explore, bigfiles, dryrun, applying, report
  docs/ 00-PLAN.md 01-ARCHITECTURE.md 02-SCANNER-PERF.md 03-SAFETY-MODEL.md 04-CATALOG.md STATUS.md
  Makefile                    build, universal, linux, test, bench, nonet-check
```

## Scanner (the FAST part)
1. **Catalog first (instant).** On start, size about 60 known locations in parallel, so the Suspects screen fills in under a second:
   - Caches: `~/Library/Caches`, npm, pnpm, yarn, pip, uv, Go build/mod, cargo registry, gradle, CocoaPods, Homebrew.
   - Xcode: DerivedData, Archives, iOS DeviceSupport, simulators.
   - Docker/OrbStack/UTM/Parallels disk images, `~/Downloads`, `~/.Trash`.
   - Time Machine local snapshots (from `tmutil`, read-only listing).
2. **Full parallel walk of the root (default `$HOME`), in the background.**
   - Pool of `4×NumCPU` goroutines with a shared channel of directories, closed when an atomic pending counter reaches zero.
   - Each directory is opened once. On macOS, `getattrlistbulk` asks for name, type, allocated size, inode, link count, mtime and device, so one call returns many entries. On Linux, `getdents64` plus `fstatat(dirfd, name, AT_SYMLINK_NOFOLLOW)`.
   - Sizes are allocated bytes. We stay on the root's device (no crossing filesystems or firmlinks), never follow symlinks, and count a file with link count > 1 only once (dev+inode set).
   - Detectors: when a directory's name matches a pattern (`node_modules`, `target` next to a `Cargo.toml`, `.venv`, `__pycache__`, `Pods`, `.next`, `build` next to a gradle file…), it becomes a suspect. Its subtree is still sized, but it's flagged so the explorer doesn't expand it by default.
3. **Memory.** Keep a node per directory (name, parent index, allocated size, file count, newest mtime) plus a global min-heap of the top 500 files over 50 MB. Don't keep individual small files. That keeps memory to tens of MB even with about 3M inodes.
4. **Streaming.** Workers send progress (directory count, bytes, current path) about every 100 ms and post sizes upward at the end. The UI renders partial results while the scan runs.
5. **Cache.** Save the snapshot when a scan finishes. On launch, show it immediately with a "N days old · rescanning" badge.
6. **`--bench`.** Runs the generic and fast readdir paths and `du -sk` on the same root, cold and warm, and prints files/s and wall time. Target: full `$HOME` scan under 10 s warm on a recent Apple Silicon laptop. The result goes in `02-SCANNER-PERF.md`.

## Safety model
- **No network.** `make nonet-check` fails if `go list -deps ./...` includes `net`, `net/http` or similar. It runs as part of `make test`.
- **What Apply accepts.** Only an Action built from a scanned node, and the path must be under the scan root.
- **Denylist** (always refused):
  - `/`
  - `$HOME` itself and its top-level folders (`~/Documents`, `~/Desktop`, `~/Library`… the folders themselves, not what's inside them)
  - `/System`, `/usr`, `/bin`, `/sbin`, `/etc`, `/private/var/db`, `/Applications` itself
  - any mount point
- **Re-check before deleting.** Right before deleting, `lstat` the path again. It must still have the same dev, inode and type as at scan time, otherwise skip it and report it.
- **Deletion itself.** Use `unlinkat` with no symlink following (`os.RemoveAll` is fine; it doesn't follow symlinks). Trash on macOS uses `osascript -e 'tell app "Finder" to delete POSIX file …'` or a move to `~/.Trash`. Linux uses the XDG Trash spec.
- **Native commands** where `rm` is wrong. Docker: `docker system prune -a`. Homebrew: `brew cleanup -s`. Simulators: `xcrun simctl delete unavailable`. Time Machine: `tmutil thinlocalsnapshots /`. They are shown verbatim in the dry run and only run after confirmation.
- **sudo:**
  - Never elevates itself.
  - Not running as root: items failing an access check get a 🔒 badge and go into a "needs sudo" section with a copyable script. The tool doesn't run those itself.
  - Running as root: detect `SUDO_USER`, use that user's home and cache dir (chown cache files back to them), show a red ROOT banner, and require typing `yes` to Apply.
  - Note in the UI: sudo doesn't get past macOS privacy protection. Count EPERM directories and suggest granting Full Disk Access to the terminal app.
- **Deletion log.** Every applied action is logged with timestamp, path, size and outcome to `~/Library/Logs/mr-plow/actions.log` (or `$XDG_STATE_HOME`).
- **Tests.** Unit tests on temp directories: denylist; symlink pointing outside the root (only the link is removed); path swapped between scan and apply (skipped); hardlink counting; mount boundary; root-mode home resolution.

## UI flow (Bubble Tea)
- **Header:** volume gauge (used/free/total, plus purgeable and snapshot hints where available), scan progress line, estimated reclaimable total.
- **Tabs:**
  - **Suspects** (default): groups Dev artifacts / Caches / VMs & containers / Old big files / Media & personal / Trash & Downloads. Each group has a bar; each item has a size, age ("untouched since 2019"), a one-line explanation, and a safety label (Safe / Regenerable / Review / Personal).
  - **Explore**: ncdu-style tree with size bars and percentage of parent; `enter` drills in, `backspace` goes up.
  - **Big files**: the top-file heap with type/extension and last modified date.
- **Keys:** `space` select, `t` toggle Trash for an item, `d` remove → **Dry run** screen.
- **Dry run screen:** exact command per item, "will free X across N items", the 🔒 sudo section, and the keys `a` Apply, `e` export as `mr-plow-plan.sh`, `esc` back.
- **Apply screen:** progress bar per item, then a **Report**: freed vs estimated, failures, disk gauge before → after.
- Colours use `lipgloss.AdaptiveColor`, so they work on light and dark terminals.

## Milestones (update STATUS.md after each)
- **M0** Scaffold, `.tool-versions`, go.mod, Makefile, docs 00–04 + STATUS.
- **M1** Scanner (generic, linux, darwin readdir), tree model, `--bench`; record numbers.
- **M2** Catalog + detectors + disk usage.
- **M3** UI shell: header, Suspects tab, live progress.
- **M4** Explore + Big files tabs.
- **M5** Plan/dry run/apply/Trash/export script, safety package + tests.
- **M6** Snapshot cache, sudo/root handling, Full Disk Access/EPERM reporting, i18n en/es.
- **M7** Release: universal macOS binary, static Linux binary, README with install steps.

## Verification
- `make test`: unit tests plus the network-import check.
- `mr-plow --bench --root ~`: compare against `du -sk ~` (sizes within about 1%, wall time recorded).
- Fixture tree script (`scripts/mkfixture.sh`): fake node_modules, an ISO-sized sparse file, symlinks pointing outside, hardlinks. Run the TUI against it with `--root`, select items, check the dry-run text, Apply, confirm the right things are gone and nothing outside was touched.
- Drive the TUI in tmux (`tmux send-keys`, `capture-pane`) to check the screens.
- Real-home run in **dry-run/export only**. Nothing on a real disk is deleted during development; Apply on real data is the user's to press.
- `file`/`lipo -info` on the release binary (universal, about 5–8 MB), and `otool -L` shows only libSystem.

---

## Deviations from the plan (decided while building)

- **No cgo.** `getattrlistbulk(2)` is called through Go's raw `syscall.Syscall6` (libSystem's `syscall` trampoline), so the macOS build is pure Go and cross-compiles. The universal binary is two `CGO_ENABLED=0` builds merged with `lipo`.
- **The "<10 s for all of $HOME" target was missed.** The real number is about 17–20 s for a ~2M-file home on the reference laptop. It's bound by APFS contention inside the kernel, not by mr-plow (see `02-SCANNER-PERF.md`). Two mitigations make it feel instant: catalog locations are scanned first, and the last scan is shown from cache while the fresh one runs.
- **Trash on macOS** is a `rename` into `~/.Trash`, not Finder AppleScript. That's simpler, needs no automation permission, and works the same under sudo. "Put Back" isn't available for these items.
- **`--report`** (text output, no UI) and **`MR_PLOW_DRY_RUN=1`** were added for scripting and safe testing.
