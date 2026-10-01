# 02 · Scanner performance

Reference machine: an 8-core Apple Silicon laptop, APFS, warm cache, in normal use during the runs, so expect ±15% noise. Your numbers will differ; run `make bench`.

## Listers

| lister | how | code folder (~460k files, ~70k dirs) | whole home (~2M files, ~280k dirs) |
|---|---|---|---|
| **getattrlistbulk** (default on macOS) | one syscall returns name, type, alloc size, inode, mtime, link count and mount status for many entries | **1.8 s** | **17.1 s** |
| readdir+fstatat (default on Linux) | open dir once, `fstatat(dirfd, name)` per entry | 2.8 s | 23.5 s |
| readdir+lstat (baseline) | `os.ReadDir` + `os.Lstat(fullpath)` | 2.8 s | 23.0 s |
| `du -sk` | single-threaded fts | 7.0 s | 62.7 s |

Totals agree with `du -sk` to within 0.2% on both trees. `scripts/sizecmp.go` compares every subtree between two listers when in doubt.

## Scaling: why not faster

Worker count against wall time for the whole home with the bulk lister:

| workers | 1 | 2 | 4 | 6 | 8 | 16 | 32 | 64 |
|---|---|---|---|---|---|---|---|---|
| time | 63.7 s | 32.9 s | 22.9 s | 15.9 s | 19.9 s | 19.5 s | 18.1 s | 19.5 s |

Scaling is linear to 2 workers, about 2.8× at 4, then flat. During the scan the process uses about 2.6 cores, almost all of it system time, whether the work queue is the original "spawn or recurse inline" or the current shared LIFO stack. The ceiling is lock contention inside APFS, not the walker. One worker runs at exactly `du` speed. The default stays at `4 × NumCPU`, which is harmless and helps on faster storage and on Linux.

## What makes it *feel* instant anyway

1. **Catalog-first ordering.** The work queue is a LIFO. Directories that are ancestors or descendants of known catalog locations are pushed last, so they're listed first. On a cold start most known garbage is on screen within ~5 s; the rest fills in while you browse.
2. **Snapshot cache.** Every finished home scan is saved (a few MB). The next launch shows the full picture in **0.6 s**, with "showing last scan from …" and a percentage estimated from the previous file count, then swaps to the fresh results.
3. **Streaming.** Sizes are live atomics. Items still being scanned show `≥ size`.

## Reproduce

```sh
make bench                    # whole home
ROOT=~/code make bench
go run scripts/workers.go ~ 1 2 4 8 16 32
go run scripts/sizecmp.go ~   # per-subtree differences between listers
```
