# Phase 3 — `branch` and `worktree`

## Goal

The first two real commands, proving the phase-2 framework end to end: `gint branch` (`br`) and `gint worktree` (`wt`).

## Scope

### `branch`

- Columns: branch name, last commit, last author, relative date. Reference output: `git-br.py`.
- Sort: last commit, created, author, off. Filters: author; date buckets (1d, 3d, 1w, 1m, YTD, 1y); merged/not-merged into current; "gone" upstreams.
- Create entry above the list (focus stays on first branch); `Shift+N` and `-b/--new`.
- Enter menu: checkout, delete (confirm; force-delete types `force`), pull, push, rename, copy sha (own shortcut).
- Select mode bulk ops: archive (tag `archive/<name>` + delete), delete (type `delete all`), force delete (type `force delete`).
- `Shift+M` merge selected into current — stub the confirmation flow now, swap in the real `merge` flow in phase 6.
- Flags: common set plus `-b/--new`, `-D/--delete`, `-m/--rename`.
- `gint branch <name>`: operations menu for that branch with fuzzy-matched input (`pull` vs `Push` case disambiguation).

### `worktree`

- Columns: shortest path (relative, or `~`-absolute), branch, commit, relative date.
- Operations: checkout/cd, search, sort, filter, delete, fetch (`f`), pull, push, rename branch, create (`Shift+N`), prune, lock/unlock, copy path.
- Same shortcuts and flags as `branch`.

## Tasks

1. Branch data source + columns + sorts/filters over the phase-1 git layer.
2. Branch operations + confirmations; copy-sha shortcut.
3. Create-branch entry row; bulk archive/delete/force-delete.
4. `gint branch <name>` direct-menu mode with fuzzy op matching.
5. Worktree data source, operations (incl. cd-on-checkout mechanism — print path / shell integration), prune, lock/unlock.
6. teatest + git-layer tests for both commands; `-I` tabular output matches reference columns.

## Acceptance

- Both commands usable daily against a real repo; destructive ops always confirm; `make lint test` clean.

## References

- PROMPT.md → `branch`, `worktree`; `~/src/env/src/common/git/git-br.py`.
