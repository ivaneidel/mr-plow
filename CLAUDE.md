# mr-plow

Disk-cleanup TUI in Go (Bubble Tea). Read `docs/STATUS.md` first, then `docs/01-ARCHITECTURE.md` and `docs/03-SAFETY-MODEL.md`.

- Run `make test` before claiming anything works. It includes the no-network check and vet for Linux.
- **Never apply deletions on the real home while developing.** Use `make fixture` + `HOME=/tmp/mr-plow-fixture/home`, or `MR_PLOW_DRY_RUN=1` / `--report` on the real home. Never run native-command actions (docker/brew/go/xcrun/tmutil) in tests.
- Any new deletion path must go through `plan.CheckPath` + the inode re-check in `apply.one`, and needs a test in `internal/apply`.
- Catalog changes: run `go run scripts/catalogdoc.go > docs/04-CATALOG.md`.
- Strings go through `internal/i18n` (en + es, Rioplatense voseo).
- Keep `docs/STATUS.md` updated per milestone.
