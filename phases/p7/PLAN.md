# Phase 7 — `rebase` and the conflict-resolution component

## Goal

The most stateful command: direct interactive rebase, plus the reusable conflict-resolution component that PROMPT.md requires to be shared with the future `resolve-conflicts` command and `status`.

## Scope

### Conflict-resolution component (build first, reusable)

- Walks conflicts per file: take ours / take theirs / take both / open in `$EDITOR`, then continue.
- Always labels the two sides with their branch/commit names — during a rebase git's "ours"/"theirs" are inverted vs a merge, so never show raw labels.
- Abort (with confirmation) and skip available at any point.
- Exposed as a component with an operation-context input (rebase/merge/cherry-pick) so `status` (phase 5 interface) and the future `resolve-conflicts` command can drive it.

### `rebase` (`reb`)

- Pick operations per commit (pick/squash/edit/drop/reword-style choices), then a final **submit** step with confirmation before anything runs.
- In-progress detection: if a rebase is already running, show its state (branch, onto, current commit, progress e.g. 3/7) and offer: resolve conflicts (if any), continue, skip, edit current commit, abort (with confirmation) — never start a second rebase.
- On conflict stop, enter the conflict-resolution component.

## Tasks

1. Conflict model: parse conflicted files/hunks, side labels resolved to branch/commit names per operation type.
2. Resolution component (ours/theirs/both/$EDITOR walk-through) + abort/skip.
3. Rebase planning view (per-commit op selection + submit confirmation), driving `git rebase` non-interactively (todo-list generation via `GIT_SEQUENCE_EDITOR`).
4. In-progress state reader (`.git/rebase-merge` / `rebase-apply`) + resume UI.
5. Wire the component into `status` conflict handling (phase-5 interface).
6. Tests: conflicted rebase fixtures, side-label correctness in rebase vs merge, resume states.

## Acceptance

- Full rebase with conflicts completable inside `gint reb`; labels always branch/commit names; component consumed by `status`; `make lint test` clean.

## References

- PROMPT.md → `rebase`, `resolve-conflicts` (future), `status` conflict hand-off.
