# Phase 3 progress

Status: **done**

- [x] 1. Branch data source, columns, sorts, filters
- [x] 2. Branch operations + confirmations
- [x] 3. Create-branch row; bulk archive/delete/force-delete
- [x] 4. `gint branch <name>` direct-menu mode
- [x] 5. Worktree command (data source, ops, prune, lock/unlock)
- [x] 6. Tests (teatest + git layer); `-I` output verified

## Log

- 2026-07-10: Implemented full phase 3 scope — `branch` and `worktree`.
  - **`internal/git`**: extended `branch.go` with `MergedBranches`,
    `BranchExists`, `RevParse`, `CreateBranch`, `CheckoutBranch`,
    `DeleteBranch`, `RenameBranch`, `PullBranch`, `PushBranch`, `TagRef`; the
    `Branch` struct grew `CommitUnix`/`AuthorUnix`/`UpstreamTrack` for
    sort/filter and `Gone()`. Extended `worktree.go` with `AddWorktree`,
    `RemoveWorktree`, `PruneWorktrees`, `LockWorktree`, `UnlockWorktree`.
  - **`internal/cli/branch.go`** (full rewrite of the phase-1 stub):
    filters (author, date bucket, merged/not-merged, gone), sorts
    (last-commit/created/author/off), a pinned create-branch row with cursor
    starting past it, the full operation registry (checkout, delete with
    force-typed confirm, pull, push, rename, copy-sha, merge-into-current
    stub for phase 6, bulk archive/delete/force-delete), and
    `gint branch <name>` direct-menu mode (`OpenMenuOnStart`).
  - **`internal/cli/worktree.go`** (new): columns (shortest relative-or-`~`
    path, branch, short sha, relative date via a per-row `git show -s
    --format=%cr`), operations (checkout/cd, delete, fetch/pull/push scoped
    to that worktree's directory via a path-scoped `Runner`, rename branch,
    copy path, lock/unlock, create, prune, bulk delete). `checkout` prints
    the path after the program exits rather than trying to `cd` the parent
    shell — see `.context/decisions.md`.
  - **Tests**: `internal/git` gained fixture-repo tests for every new
    branch/worktree function (exists/create/checkout/delete incl. the
    force-required-on-unmerged case, rename, merged-set, revparse+tag,
    add/remove/lock/unlock/prune). `internal/cli` gained unit tests for the
    pure helpers (`filterBranches`, `sortBranches`, `sortModeFromFlag`,
    `branchFilters.validate`, `shortestPath`, `shortSHA`); the interaction
    flows themselves (menu, confirm, select-mode) stay covered by phase 2's
    generic `internal/tui` teatest suite, which every `Operation`-driven view
    shares.
  - Verified acceptance: `./gint branch -I` / `-I -F` and `./gint worktree
    -I` / `-I -s` print the expected columns against this repo;
    `make lint test` clean (0 lint issues, all tests pass).
