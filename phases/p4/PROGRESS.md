# Phase 4 progress

Status: **done**

- [x] 1. Log data source
- [x] 2. Log view + cherry-pick/squash/reset flows
- [x] 3. Multi-commit cherry-pick (select mode)
- [x] 4. Graph renderer
- [x] 5. `graph` + `graph-branch` commands with `-A`
- [x] 6. Tests

## Log

- 2026-07-10: Implemented full phase 4 scope — `log`, `graph`, `graph-branch`.
  - **`internal/git`**: extended `log.go`'s `Commit` with `SHA` (full,
    for checkout/cherry-pick/reset) and `AbsDate` (ISO 8601, for `-F`);
    `logFormat` grew `%H` and `%cI`. Added `CheckoutCommit` (detached),
    `CherryPick` (multi-sha, `--no-commit` option), `SquashHead` (soft
    reset + amend — HEAD only, see `.context/decisions.md`), `ResetTo`
    (soft/mixed/hard). New `graph.go`: `ListCommitGraph` shells out to
    `git log --graph`, splitting each line's ASCII graph glyphs from its
    commit data via a sentinel byte prepended to the pretty-format string
    (decision recorded in `.context/decisions.md`); `simplify` toggles
    `--simplify-by-decoration` for `graph-branch`'s "last commit per branch"
    view.
  - **`internal/cli/log.go`** (replaces the phase-1 stub): columns (sha,
    message, date, author-as-initial, local branches, worktree dir via a
    branch→worktree-path lookup over `git.ListWorktrees`), `-F` swaps
    relative date/initials for absolute date/full name and removes the
    message width cap. Operations: checkout, copy-sha, cherry-pick (bulk,
    no/yes/no-commit confirm), squash (HEAD-only), reset
    (no/soft/mixed/hard, hard typed), and a merge-into-current stub for
    phase 6. Branch's delete/rename/pull/push are deliberately not offered
    (decision recorded).
  - **`internal/cli/graph.go`** (replaces both `graph`/`graph-branch`
    stubs): shared `runGraph(simplify bool)` builds both commands' `RunE`;
    `-A/--not-all` bases the graph on HEAD instead of every branch. The
    graph-glyph column is excluded from `FilterValue` so `/` search only
    matches commit content, and connector-only rows (no commit) never match.
  - **Tests**: `internal/git` gained fixture-repo tests for
    `ListCommitGraph` (incl. `--simplify-by-decoration` actually dropping
    an undecorated commit) plus a unit test for `parseGraph`'s connector-line
    handling, and tests for every new commit-mutation function
    (checkout/cherry-pick/squash/reset). `internal/cli` gained unit tests
    for `authorInitial` and the pluralization helper; interaction flows
    (menu, confirm, bulk cherry-pick selection) ride on phase 2's generic
    `internal/tui` teatest suite, same as phase 3.
  - Verified acceptance: `./gint log -I` / `-I -F`, `./gint graph -I`, and
    `./gint graph-branch -I` all render correctly against this repo's real
    history; `make lint test` clean (0 lint issues, all tests pass).
