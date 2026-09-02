# Phase 14 progress

Status: **complete**

- [x] 1. `tui/format.go` — `UltraBranch`, `FormatBranch` ultra-short dispatch, `FormatWorktreePath` (shortest/relative/absolute), hidden-column maps + getters
- [x] 2. `tui/settings.go` — `WorktreePathFormat`/`BranchHiddenColumns`/`LogHiddenColumns` + normalization + `CurrentSettings`
- [x] 3. `tui/item.go` — `FilterColumns` helper
- [x] 4. `tui/list.go` — `Config.ColumnsRefresh` + `View`, `SetColumns`, `applyColumnsRefresh`
- [x] 5. `tui/settings_model.go` — view-aware generic row list (cycle/toggle/theme), branch+log Display sections, worktree-path row, esc-revert + live column refresh
- [x] 6. `gint lg` — `branchesCell` (FormatBranch over refs), raw worktree paths formatted live, FilterColumns + ColumnsRefresh
- [x] 7. `gint br` — worktree column (5th cell, `DensityNormal`), create-row padding, FilterColumns + ColumnsRefresh; `-I`/direct-menu unfiltered
- [x] 8. `gint wt` — path column honors the same worktree-path format; `shortestPath` removed
- [x] 9. Tests — format, settings, overlay (sections/toggle/revert/save), FilterColumns, branch/log items, existing settings-overlay tests green
- [x] 10. Docs — p14 phase files, `phases/README.md`, `PROMPT.md` Settings section, `.context/decisions.md`

## Log

### 2026-08-29

- Phase created. Added per-view column toggles (`:settings` → Display) to `gint br` and `gint lg`, a **worktree** column to `gint br` (sourced from `git worktree list`, so the main worktree resolves too), a global **worktree-path format** (shortest/relative/absolute) shared by branch/log/worktree views, and a third **ultra-short** branch format (last segment, vowels stripped). Hiding/formatting is TUI-only: `-I`/`RenderTable` keeps the full column set. `make lint test` green (race detector on).
- **Note**: the plan's branch worktree column snippet said `DensityFull`, but its prose said "visible in default and -F, hidden in -s" — the prose is the intent, so the column uses `DensityNormal`. The settings-overlay `newSettingsModel` takes the owning `*List` and reads its `View` (internal signature differs trivially from the plan's two-arg form).
- **Bug fix (user-found)**: the plan's architecture for hiding — cli filters the `[]Column` slice via `FilterColumns` before `tui.New` — was wrong. `columnIndex` maps a visible column back to its position in `l.columns`, and `cell()` indexes item cells positionally against that same set; pre-filtering desynced the two, so rows rendered the wrong cells (branch names under the `last commit` header). Fix: `l.columns` is now always the full set; hiding applies inside `visibleColumns()` via a live `Config.HiddenColumns func() map[string]bool` predicate re-evaluated every frame. `FilterColumns`, `ColumnsRefresh`, and `SetColumns` were removed — toggles preview live for free, same as the format vars. Regression test: `TestHiddenColumnsKeepCellAlignment`.

## Verification

- `make lint test` — 0 lint issues; all packages pass with `-race`.
- Manual (`gint br -I` / `gint lg -I` in a repo with a linked worktree): worktree column shows `.` for the main checkout and `~/…` for linked worktrees; `-s` hides it; `branchFormat: ultra-short` renders `lgn-frm`/`mn`; `worktreePathFormat: absolute|relative` render full/relative paths (relative escapes cwd → absolute fallback).
- Settings persistence: toggling Display rows + cycling worktree path in the overlay then `s` writes the new keys to `settings.json` (covered by `TestSettingsModelSavePersistsNewFields`).