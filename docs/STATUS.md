# STATUS

_Last updated: 2026-10-01._

## Milestones

| | milestone | state |
|---|---|---|
| M0 | scaffold, `.tool-versions`, Makefile, docs | ✅ |
| M1 | scanner: getattrlistbulk / fstatat / generic listers, tree, `--bench` | ✅ numbers in 02-SCANNER-PERF |
| M2 | catalog (macOS + Linux), detectors, disk usage | ✅ |
| M3 | UI shell, header graphs, Suspects tab, live progress | ✅ |
| M4 | Explore + Big files tabs | ✅ |
| M5 | plan / dry run / apply / Trash / export script, safety + tests | ✅ |
| M6 | snapshot cache, sudo handling, Full Disk Access reporting, i18n en/es | ✅ (sudo not run live; see below) |
| M7 | release builds: universal macOS, static Linux, README | ✅ |

## Verified

- `go test -race ./...` passes: scanner fixture (symlink not followed, hard link once, nested `node_modules` tagged once, every lister agrees), snapshot round trip, denylist, apply guards (symlink inside a deleted folder, path swapped for a symlink between scan and apply, dry run, nested selections, blocked paths, Trash, empty-folder mode), session items on a fake home (nvm's `node_modules` excluded).
- Full UI flow driven in tmux against `scripts/mkfixture.sh`, with a `-race` binary: scan → rescan → select → dry run → apply → report → language toggle → explore → quit. No races. The filesystem diff matched exactly what was approved, and the files outside were untouched.
- A real home folder (~2M files) in **read-only** modes only (`--report`, `MR_PLOW_DRY_RUN=1`): full scan in about 20 s, cache reload in under a second. Nothing on a real disk was deleted during development.
- `dist/` binaries: universal Mach-O (x86_64 + arm64; the Intel build was run under Rosetta), Linux ELF static. `make nonet-check` passes.

## Not verified / open

- **sudo / root mode** has never been run under a real `sudo`. Try `sudo bin/mr-plow --dry-run` once and check:
  - the home shown is yours
  - `~/Library/Caches/mr-plow` is owned by you afterwards
- **Linux** is cross-compiled and vetted, but has never been run on a Linux machine.
- **Native commands** (`docker system prune`, `brew cleanup`, `go clean -modcache`, `xcrun simctl delete unavailable`, `tmutil thinlocalsnapshots`, `pnpm store prune`) were never executed during development, because they act on the real system. Only their rendering in the dry run was checked.
- Full-home scans take about 17–20 s, against a target of under 10 s. It's kernel-bound (02-SCANNER-PERF). An idea for later: macOS FSEvents history (`sinceWhen`) to rescan only the folders that changed since the cached snapshot.

## Next ideas

- FSEvents incremental rescans (see above).
- Sizes for Time Machine local snapshots (`diskutil apfs listSnapshots` doesn't give sizes; would need `tmutil` heuristics).
- A "last touched" filter on detector items (e.g. only `node_modules` untouched for 6+ months).
