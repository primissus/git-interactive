# Phase 14 — Per-view column settings + worktree column

## Goal

Make each view's columns user-configurable: `gint br` and `gint lg` gain a **Display** section in their `:settings` overlay that toggles which columns render, plus per-view format rows (branch format incl. a new **ultra-short** value; author format in log). `gint br` gains a **worktree** column showing where each branch is checked out, and a global **worktree-path format** setting (`shortest`/`relative`/`absolute`) is shared by branch, log, and worktree views.

## Decisions

- **Persistence**: per-view hidden-column lists + worktree path format persist to `settings.json` (`branchHiddenColumns`, `logHiddenColumns`, `worktreePathFormat`). **TUI only** — `-I`/`RenderTable` output always keeps the full column set.
- **lg branches column**: kept, now toggleable, and each ref is run through `FormatBranch` so the branch format setting applies there too.
- **Worktree cell** = the worktree's path (not a marker). Sourced from `git worktree list` (covers the main worktree; `%(worktreepath)` alone is empty there). One global `WorktreePathFormat`: `shortest` (default, the historical behavior) | `relative` | `absolute`.
- **ultra-short** becomes a third global `BranchFormat` value: the last `/`-segment with vowels stripped (e.g. `feat/auth/login-form` → `lgn-frm`). It affects every view that renders branch names — same as the existing `short`.
- **Settings overlay becomes view-aware**: generic sections (appearance, date format, themes) stay in every view; the `branch` view adds a Display section + worktree-path row; the `log` view adds a Display section + author/branch format rows. Other views keep today's menu.
- **Column hiding lives inside the List**: `l.columns` stays the full set (item cells index into it positionally); `visibleColumns()` drops hidden titles via a live `Config.HiddenColumns` predicate, so toggles preview without any column-set swapping. *(Plan revision: the original cli-layer `FilterColumns` approach broke the cell↔column positional invariant — see PROGRESS.md.)*

## Scope

### `tui/format.go`
- `UltraBranch(name)`: last `/`-segment with `aeiouAEIOU` removed.
- `FormatBranch` dispatches `full`/`short`/`ultra-short`.
- `activeWorktreePathFormat` + `FormatWorktreePath(path, cwd)` (`shortest`|`relative`|`absolute`; `relative` falls back to absolute when the rel path escapes cwd).
- `freezeFormats` gains the worktree-path format + hidden-column maps (`activeBranchHidden`, `activeLogHidden`); exported getters `ActiveBranchHidden()`, `ActiveLogHidden()`, `ActiveWorktreePathFormat()`.

### `tui/settings.go`
- `Settings` gains `WorktreePathFormat`, `BranchHiddenColumns`, `LogHiddenColumns` (omitempty).
- `LoadSettings` normalizes branch format (`full|short|ultra-short`) and worktree path format (`shortest|relative|absolute`).
- `CurrentSettings()` returns the new fields.

### `tui/item.go`
- `FilterColumns(cols, hidden)` drops hidden titles, preserving order, never mutating the input.

### `tui/list.go`
- `Config` gains `ColumnsRefresh func() []Column` and `View string`; `List` exposes `SetColumns` + `applyColumnsRefresh`.

### `tui/settings_model.go`
- Generic row list (`settingsRow` kinds: cycle / toggle / theme); view-aware sections.
- `branch` view: Display toggles for branch/last commit/date/author/worktree + worktree-path cycle row.
- `log` view: Display toggles for sha/message/date/author/branches/worktree + author + branch format rows.
- Save persists new fields; Esc reverts toggles + formats live.

### `tui/keymap.go`
- New chrome fields `SettingsDisplay`, `SettingsWorktreePath` (+ override plumbing).

### `cli/branch.go`
- `branchItem` gains a 5th worktree cell (formatted live); `createBranchItem` matches.
- `branchColumns()` gains `worktree` column (`Density: DensityNormal` — visible in default/-F, hidden in `-s`).
- `loadBranchItems` attaches each branch's worktree path (via the `git worktree list` map).
- Interactive list filters columns via `FilterColumns(branchColumns(), tui.ActiveBranchHidden())` + `ColumnsRefresh`; `-I` and direct-menu keep unfiltered columns.

### `cli/log.go`
- Branches cell runs each ref through `FormatBranch` (`branchesCell`).
- `worktreeByBranch` returns raw absolute paths; cells format via `FormatWorktreePath`.
- Interactive list filters via `FilterColumns(logColumns(...), tui.ActiveLogHidden())` + `ColumnsRefresh`.

### `cli/worktree.go`
- `gint wt` also formats its path column via `FormatWorktreePath` (one global setting); `shortestPath` removed.

### Docs
- `phases/p14/{PLAN,PROGRESS}.md`, `phases/README.md`, `PROMPT.md` (Settings section), `.context/decisions.md`.

## Tests

- `tui/format_test.go`: `UltraBranch` table (multi-segment, no-slash, vowel-only, mixed case), `FormatBranch` dispatch, `FormatWorktreePath` (3 formats + escapes), `freezeFormats` new fields, hidden-set helpers.
- `tui/settings_test.go`: new-field load/round-trip/fallback, branch/log overlay sections, toggle live-preview + esc-revert, worktree-path cycle, save-persistence.
- `tui/table_test.go`: `FilterColumns` (none/one/all hidden, input unmutated).
- `cli/branch_test.go`: worktree cell present; hidden column dropped from filtered set but not from `-I` RenderTable.
- `cli/log_test.go`: `branchesCell`, log worktree cell formatting.
- Existing `list_test.go` settings-overlay tests stay green (demo view unchanged).

## Files touched

`internal/tui/{format.go, settings.go, item.go, list.go, settings_model.go, keymap.go}` + their tests, `internal/cli/{branch.go, log.go, worktree.go}` + their tests, `phases/p14/{PLAN,PROGRESS}.md`, `phases/README.md`, `PROMPT.md`, `.context/decisions.md`.