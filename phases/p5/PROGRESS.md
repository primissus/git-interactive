# Phase 5 progress

Status: **done**

- [x] 1. Status data source (porcelain v2, conflict states)
- [x] 2. Status view (stage toggle, diff, discard, stash file)
- [x] 3. Conflict continue/abort behind hand-off interface
- [x] 4. `add` command
- [x] 5. `stash` command
- [x] 6. Tests on dirty/conflicted fixtures

## Log

- 2026-07-10: Implemented full phase 5 scope — `status`, `add`, `stash`.
  - **`internal/git`**: extended `status.go` with `StageFile`/`UnstageFile`/
    `StageAll`/`UnstageAll`/`DiscardFile`/`CleanUntracked`/`DiffFile`;
    extended `stash.go` with `StashPush` (optional path scoping for the
    per-file stash operation)/`StashApply`/`StashPop`/`StashDrop`/
    `StashClear`/`StashDiff`. New `conflict.go`: `InProgressState` — detects
    an in-progress merge/rebase/cherry-pick/revert from `.git`'s marker
    files (`MERGE_HEAD`, `rebase-merge`/`rebase-apply`, `CHERRY_PICK_HEAD`,
    `REVERT_HEAD`) and offers `Continue`/`Abort`. Kept independent of any
    TUI code so phase 7's `resolve-conflicts` (PROMPT.md → Future) can reuse
    the same detection — decision recorded in `.context/decisions.md`.
  - **`internal/cli/status.go`** (replaces the phase-1 stub): merges
    staged/unstaged/untracked/conflict entries into one deduped list.
    Operations: stage/unstage toggle (bound to `t`, not Enter — the shared
    `List` always uses Enter for its context menu; decision recorded),
    diff (`d`, piped through `$PAGER`/`less` via `tea.ExecProcess`, new
    `internal/cli/pager.go` shared with `stash`'s diff), discard (typed
    `discard`), per-file stash, and commit/amend stubs for phase 6. When
    `git.DetectInProgress` finds an in-progress operation, continue/abort
    operations are appended.
  - **`internal/cli/add.go`**: reuses `status.go`'s data source/columns
    with a staging-only operation set (stage/unstage incl. bulk, stage
    all/unstage all, restore file, clean untracked — typed `clean`).
  - **`internal/cli/stash.go`**: index/message/branch/date columns,
    apply/pop/drop/diff/create(with message input)/clear-all (typed
    `clear all`).
  - **Tests**: `internal/git` gained fixture-repo tests for every new
    status/stash mutation function and for `DetectInProgress` (incl. a real
    merge-conflict-then-abort round trip). `internal/cli` gained a unit
    test for `fullyStaged`; interaction flows ride on phase 2's generic
    `internal/tui` teatest suite.
  - Verified acceptance: `./gint status -I`, `./gint add -I`, and
    `./gint stash -I` render correctly against this repo's real (dirty)
    working tree; `make lint test` clean (0 lint issues, all tests pass).
