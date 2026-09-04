# Phase 15 progress

Status: **complete**

- [x] 1. `tui/operation.go` + `tui/list.go` — `Operation.ConfirmFrom`, resolved in `runConfirm` ahead of a static `Confirm`
- [x] 2. `internal/gh/{gh.go,gh_test.go}` — `Available`, `PR`, `ListPRs`/`PRsByBranch`, `OpenPR`, `parsePRs`
- [x] 3. `tui/item.go` + `tui/search.go` — `Searchable` interface, `filterItems` prefers it over `FilterValue`
- [x] 4. `cli/worktree.go` — `worktreeItem.SearchValue`, `pr` column + cell, `worktreeViewState` (mutex-guarded PR cache), `worktreePRInitCmd`, "open pr" op
- [x] 5. `cli/branch.go` — `branchItem.SearchValue`/`pr` field/cell, `checkoutConfirm` + checkout's `ConfirmFrom`, `--cd-file` flag, `branchViewState` moved behind `sync.Mutex` with copy-on-write accessors (`toggleGrouped`, `toggleCollapsed`, `setPRCache`, `snapshot`, `rebuild`), `prInitCmd`, "open pr" op
- [x] 6. `cli/shellinit.go` — POSIX + fish wrappers match `branch`/`br` alongside `worktree`/`wt`
- [x] 7. `cli/branch_tree.go` — `treeNode.insert` keeps the full item (was dropping `wtPath`/`cwd`); `branchGroupItem`/`createBranchItem` padded to 6 cells
- [x] 8. `tui/settings_model.go` — `branchColumnTitles` gains `pr`
- [x] 9. Tests — `internal/gh`, `tui/list_test.go` (ConfirmFrom, InitCmd), new `tui/search_test.go` (Searchable precedence, batch labels unaffected), `cli/branch_test.go`, `cli/branch_tree_test.go`, `cli/worktree_test.go`, `cli/shellinit_test.go`, `tui/settings_test.go` row/column-count updates
- [x] 10. Docs — `PROMPT.md`, `README.md`, `.context/{decisions.md,architecture.md,known-issues.md}`, `phases/README.md`, this phase's files

## Log

### 2026-09-04

- Phase created and implemented in one pass. Branch checkout now offers to `cd` into a branch's worktree instead of failing with git's raw "already used by worktree" error; the worktree view's `/` search now matches the `~`-abbreviated path actually shown on screen (via the new `Searchable` interface, keeping `FilterValue` — and batch-failure labels — untouched); `branch` and `worktree` both gained a best-effort `pr` column and "open pr" operation backed by a new `internal/gh` package, loaded asynchronously via a new `Config.InitCmd` (`List`'s first background `tea.Cmd`) so the list renders before the `gh pr list` round trip completes.
- **Concurrency**: introducing a genuine background goroutine (the PR fetch) meant `branchViewState`'s `sort`/`grouped`/`collapsed`/`prCache` — previously plain fields mutated directly by every operation on the update goroutine — needed synchronization, since the fetch goroutine now reads them too. Used a `sync.Mutex` with small locked accessors and copy-on-write map replacement (a toggle builds a new map rather than mutating the shared one in place) so a snapshot handed to the background goroutine stays valid without its own lock. `rebaseBase`/`cdPath` stay unguarded — only the update goroutine (an operation's `Run`, and the main loop right after `p.Run()` returns) ever touches them. `go test -race ./...` is clean.
- **Verified manually**: `gint br -I`/`gint wt -I` in a repo with no GitHub remote (empty `pr` column, no delay); in this repo (a real GitHub remote, `gh` authenticated) the `pr` column renders and the `-I` round trip is ~0.7s, acceptable for a one-shot print; in a scratch repo with a linked worktree, pressing `C` on the branch checked out elsewhere shows the "already checked out at ... — move there instead?" prompt with `[n] no [c] cd there`, and pressing `C` on the current branch (no worktree conflict) checks out directly with no prompt.

## Verification

- `make fmt lint test` — 0 lint issues; all packages pass with `-race`; `make build` succeeds.
- `gh` availability gating confirmed via `TestParsePRs*`/`TestPRsByBranchKeysOnHeadRef` (no `gh` invocation in tests) plus manual runs against both a non-GitHub scratch repo and this repo's real GitHub remote.
- Tree view (`T`) manually confirmed to keep rendering the worktree and (now) `pr` columns, backed by the `TestApplyGroupingPreservesWorktreeAndPR` regression test.
