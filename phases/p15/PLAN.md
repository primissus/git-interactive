# Phase 15 — worktree-aware checkout, search audit, GitHub PR column

## Goal

Three gaps in the `branch` and `worktree` views, all reported by the user.

**Checking out a branch that lives in another worktree fails with raw git noise.** The branch view's checkout operation calls `git checkout` with no worktree awareness, even though every row already carries the owning worktree's absolute path in `branchItem.wtPath` and renders it in the worktree column. Git refuses with `fatal: 'x' is already used by worktree at '...'`, and that stderr is dumped into the one-line footer. The user wants a prompt offering to move to that folder instead.

**Worktree search needs an audit.** The user believes `/` already matches worktree and branch names and asked for confirmation rather than a rewrite. It mostly does, with one real hole described below.

**No visibility into open pull requests.** `gh` is installed but the tool never shells out to anything but `git`. The user wants a PR column in `branch` and `worktree`, loaded asynchronously, silently skipped when `gh` is unavailable, plus an operation to open the PR in a browser.

## Decisions

- **Checkout prompt offers "cd there" and "no" only.** The user explicitly narrowed the original "or also remove the worktree" idea to just moving. Removing a worktree stays where it already lives, in `gint worktree`'s delete operation.
- **PR fetch is async and best-effort.** No `gh` on PATH, not a GitHub remote, not authenticated, or any non-zero exit means the column stays empty and nothing is reported. No flag, no setting to enable it.
- **PR data is fetched once per view session,** not on every refresh. Refreshes reuse the cached map so the column does not blank out after a checkout or delete.
- **`gh` gets its own package** (`internal/gh`), mirroring `internal/git`'s shape: a `Runner` with `Dir`, free functions taking `(ctx, r, ...)`, and a private parser. It does not extend `git.Runner`, whose exec target is hardcoded to `git`.

See `.context/decisions.md` for the full rationale behind each of these (search: "phase 15").

## Scope

### 1. Conditional confirmation (framework)

`Operation` gains `ConfirmFrom func([]Item) *Confirm` ([operation.go](../../internal/tui/operation.go)): when set, `List.runConfirm` ([list.go](../../internal/tui/list.go)) resolves the confirmation from the operation's target items at invocation time instead of using a static `Confirm`; returning `nil` skips the prompt and runs immediately. It overrides `Confirm` when both are set.

### 2. Branch checkout into another worktree

[branch.go](../../internal/cli/branch.go): the checkout operation gets `ConfirmFrom: checkoutConfirm`, firing only when the target's `wtPath` is non-empty and the branch is not this worktree's HEAD, offering **no** / **cd there**. On "cd", `branchViewState.cdPath` is set and the view quits; `runBranch`/`runBranchDirectMenu` write that path to a new hidden `--cd-file` flag (mirroring `worktree`'s) or print it as the final stdout line. [shellinit.go](../../internal/cli/shellinit.go)'s wrapper case arm widens from `worktree | wt` to also match `branch | br` in both the POSIX and fish scripts — existing installs need a fresh `eval "$(gint shell-init zsh)"` for this to actually `cd`.

### 3. Worktree search audit

Finding: `worktreeItem.FilterValue()` returns the **raw absolute** path plus the branch, but the column renders the path through `FormatWorktreePath`. Under the default `shortest` format a sibling worktree outside the cwd displays as `~/src/wt/foo` while the filter sees `/Users/you/src/wt/foo` — a query typed against what's on screen matches nothing. Branch matching itself already worked.

Fix: a new optional `tui.Searchable` interface (`SearchValue() string`) in [item.go](../../internal/tui/item.go). `filterItems` ([search.go](../../internal/tui/search.go)) prefers it when an item implements it, falling back to `FilterValue()`. `worktreeItem` and `branchItem` implement it (raw path, formatted worktree path, branch/PR number); `FilterValue()` is untouched, since it also doubles as the human-readable label in resilient bulk operations ([batch.go](../../internal/tui/batch.go)).

### 4. `internal/gh` package

New `internal/gh/{gh.go,gh_test.go}`: `Available()`, `type PR struct{ Number, State, IsDraft, Title, HeadRefName, URL }`, `ListPRs`/`PRsByBranch` (`gh pr list --json number,state,isDraft,title,headRefName,url`, verified against `gh` 2.100.0), `OpenPR` (`gh pr view <n> --web`), and `parsePRs` as the unit-testable seam.

### 5. PR column and operation

`branchColumns()`/`worktreeColumns()` gain a `pr` column (`DensityNormal`, `tui.ColorRef`); the cell reads `#412 open`/`#412 draft`, empty with no PR. Both views gain an `open pr` operation. `"pr"` is registered in `branchColumnTitles` (a Display toggle); the worktree view has no settings section at all yet, so its `pr` column is simply always on. `-I` output fetches PRs synchronously before rendering — a one-shot print can afford the round trip and still skips entirely without `gh`.

### 6. Async loading

`tui.Config` gains `InitCmd tea.Cmd`, returned from `List.Init()` batched with the existing `textinput.Blink` direct-menu path. Skipped (`nil`) whenever `gh.Available()` is false, so no subprocess ever spawns. This is the first background `tea.Cmd` in the codebase (see decisions.md); `branchViewState` moved its `sort`/`grouped`/`collapsed`/`prCache` fields behind a `sync.Mutex` with copy-on-write accessors (`toggleGrouped`, `toggleCollapsed`, `setPRCache`, `snapshot`), since the background fetch goroutine and every operation's `Run` (on the update goroutine) now read/write them concurrently; `rebaseBase`/`cdPath` stay unguarded since only the update goroutine ever touches them. `worktreeViewState` is the equivalent, minus the sort/grouping fields the worktree view doesn't have. The background closure only ever calls `loadBranchItems`/`loadWorktreeItems` — raw-data construction — never `Item.Columns()`/`SearchValue()`/a `tui` format getter, all of which read package vars owned by the render goroutine.

### 7. Tree-view bugs surfaced by the new column

- `branch_tree.go`'s `treeNode.insert` rebuilt each leaf as `branchItem{b: b.b, merged: b.merged}`, dropping `wtPath`/`cwd`/`pr` — tree view blanked the worktree column, and would have blanked `pr` too. Fixed: append the passed-in item as-is; `flatten` still only sets `displayName` on its own copy.
- `branchGroupItem.Columns()`/`createBranchItem.Columns()` padded to 4/5 cells for what is now a 6-column table; padded to 6.

## Files touched

`internal/gh/{gh.go,gh_test.go}` (new); `internal/tui/{operation.go,list.go,item.go,search.go,settings_model.go}`; `internal/cli/{branch.go,branch_tree.go,worktree.go,shellinit.go}`; the matching `_test.go` files; `phases/p15/{PLAN.md,PROGRESS.md}`; `phases/README.md`; `PROMPT.md`; `README.md`; `.context/{decisions.md,architecture.md,known-issues.md}`.

## Tests

- `internal/gh/gh_test.go` — `parsePRs` over fixture JSON: open, draft, empty array, malformed input. No `gh` invocation.
- `internal/tui/list_test.go` — `ConfirmFrom` fires/overrides/is skipped; `InitCmd` batches with the direct-menu blink and its message reaches `setItems`.
- `internal/tui/search_test.go` (new) — `Searchable` takes precedence over `FilterValue` in `filterItems`; batch-failure labels keep using `FilterValue` even when the item implements `Searchable`.
- `internal/cli/branch_test.go` — `checkoutConfirm` fires only when `wtPath` is set and the branch isn't HEAD; the checkout op's Run stores `cdPath` and quits on `Choice == "cd"`; PR cell formatting; `SearchValue` includes the formatted worktree path and PR number while `FilterValue` stays the branch name; 6-cell `branchItem.Columns()`.
- `internal/cli/branch_tree_test.go` — grouping preserves `wtPath`/`pr`; `branchGroupItem.Columns()` returns 6 cells.
- `internal/cli/worktree_test.go` — PR cell formatting; `SearchValue` matches the `~`-abbreviated display path under the `shortest` format while `FilterValue` stays raw.
- `internal/cli/shellinit_test.go` — both wrappers match `branch`/`br` alongside `worktree`/`wt`.
- `internal/tui/settings_test.go` — `branchDemoColumns`/`branchColumnTitles` gain `pr`; row-count and visible-column assertions shift accordingly.

## Verification

1. `make fmt lint test` clean (0 lint issues, `-race` green), then `make build`.
2. In a repo with a second worktree: `gint br`, move to a branch checked out elsewhere, press `C`. The prompt appears; "cd there" with the wrapper installed lands the shell there, without it the path prints as the last line; "no" leaves the list untouched.
3. Press `C` on an ordinary branch: no prompt.
4. `gint wt`, press `/`, type the `~`-prefixed path shown in the path column: the row matches.
5. In a repo with open PRs: `gint br` shows rows immediately, then the PR column fills in; the "open pr" operation opens the right one in the browser. Same for `gint wt`.
6. `PATH= gint br` or a non-GitHub repo: no PR column content, no error, no delay.
7. Press `T` in `gint br`: the worktree and PR columns still render under tree view.
