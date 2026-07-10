# Phase 6 — `commit` and `merge`

## Goal

The commit flow and the merge flow, then wiring them into the earlier views that stubbed them (`status` commit `c`, `branch`/`log` merge `Shift+M`).

## Scope

### `commit` (`cm`)

- Message input, then a confirmation offering: no / yes / amend / stage all & commit / commit & push / no-verify / edit (back to message input).
- `Shift+A`: amend last commit — warn if the commit is already pushed (upstream contains it).
- `Shift+V`: commit with `--no-verify`, also a direct shortcut.

### `merge` (`mrg`)

- Confirmation offers: no / yes (default merge) / `--ff-only` / no-ff / squash.
- Merge in progress: continue / abort.
- On conflict, surface the conflict state (full resolution arrives in phase 7; for now show conflicted files and offer continue/abort).

### Integration

- `status` `c`/`Shift+A` open the real commit flow.
- `branch` and `log` `Shift+M` reuse the real merge confirmation flow.

## Tasks

1. Commit flow component (message input + multi-choice confirm + already-pushed warning).
2. `commit` command with amend/no-verify shortcuts.
3. Merge flow component + `merge` command, including in-progress continue/abort.
4. Replace phase-3/4/5 stubs with the real flows.
5. Tests: each confirmation branch, amend warning, merge-in-progress detection.

## Acceptance

- Commit and merge usable standalone and from status/branch/log; stubs gone; `make lint test` clean.

## References

- PROMPT.md → `commit`, `merge`.
