# 01 · Architecture

```
cmd/mr-plow           flags → session → ui (or --bench / --report)
internal/scan         parallel walker → size tree (dirs only) + top-500 big files
internal/catalog      known locations (per OS) + project-artifact detectors
internal/session      runs scans, cache, builds "Suspects" items from catalog + tree
internal/plan         selection → actions → exact shell commands (the dry run)
internal/apply        executes Ready actions, re-checking each path; action log
internal/safety       denylist, sudo/real-user identity, write-permission checks
internal/disk         statfs volume usage
internal/i18n         en/es strings (Rioplatense voseo)
internal/ui           Bubble Tea model + views
```

## Data flow

1. `session.Start` loads `~/Library/Caches/mr-plow/last-scan.gob.gz` (if the root is home), then starts the **main scan** of the root. When the root is home it also starts small **extra scans** for system catalog entries (e.g. `/Library/Caches`) and counts Time Machine local snapshots with `tmutil listlocalsnapshots /`, which is read-only.
2. The UI calls `session.View()` every 200 ms. `View` picks a snapshot: the fresh scan once it's done, else the cache, else the live partial tree. It then rebuilds the Suspects groups:
   - **Explicit entries** (`Mode Remove/Contents/Native/Info`): looked up in the tree by path. Globs are allowed.
   - **Expand entries** (`~/Library/Caches`, `~/Downloads`…): each child (or grandchild with `ExpandDepth: 2`) above `ExpandMin` becomes an item, plus big files sitting directly in that folder. A child that equals an explicit entry, or contains a native/expand entry, is skipped so the same bytes aren't offered twice.
   - **Detector items**: directories tagged during the walk (`node_modules`, `target/` next to `Cargo.toml`…), grouped per detector. Tags inside tool-managed areas are dropped: a hidden folder under home, `~/Library` or `~/Applications` anywhere in the ancestors (e.g. nvm's global `node_modules`, VS Code extensions).
   - **Old big files**: entries in the top-files heap older than a year that are disk images, archives or videos, or are over 1 GB.
3. Selecting items and pressing `d` → `plan.Build`:
   - Nested selections are merged (the parent wins).
   - Each action gets a state: **Ready**, **NeedsRoot** (no write permission or system-wide; goes to a sudo script) or **Blocked** (denylist, missing tool).
4. `a` + confirm → `apply.Run` in a goroutine, which sends progress messages to the UI. Successful removals are detached from the tree (`Dir.Detach`) so sizes update without a rescan. If the scan was interrupted by the apply, it's redone in the background while the current data stays visible.

## The size tree

`scan.Dir` holds a name, a parent pointer, dev/ino, a detector tag, and atomics: subtree size, file count, children slice (`atomic.Pointer`), done flag, pending count, newest mtime. Files aren't stored, only counted, except files of at least 50 MB, which go into a 500-entry min-heap. The UI reads the tree while workers write it:
- A worker publishes a directory's children only once they're fully built.
- Sizes propagate up the parent chain with atomic adds.
- "Done" bubbles up when a directory's pending count reaches zero.

Explore lists the *files* of the folder you open on demand, with one directory listing.

## Snapshot cache

The tree is flattened (`parent` as an index) and saved with gob + gzip: a few MB for ~300k directories. It's tagged with the root path and is only used when the root matches. Under sudo it's written into the real user's home and chowned back to them.
