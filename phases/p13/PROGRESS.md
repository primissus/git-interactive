# Phase 13 progress

Status: **complete**

- [x] 1. `tui/format.go` — ShortDate, ShortBranch, ShortAuthor, InitialsAuthor, Format* dispatch, frozen vars
- [x] 2. `tui/settings.go` — new Settings fields + normalization + freeze in ApplySettings
- [x] 3. Git data: `%ct` added to log/stash/worktree
- [x] 4. `tui/render.go` — deficit-weighted layout algorithm
- [x] 5. Column defs: ColorName + Flex on message columns + date MinWidth 7
- [x] 6. Item Columns: FormatBranch/FormatDate/FormatAuthor in all views
- [x] 7. `:settings` overlay: 3 format-option rows with live preview
- [x] 8. Chrome strings in keymap.go
- [x] 9. Tests: format, layout, settings, git parse, cli items
- [x] 10. Docs: p13 phase files, .context updates

## Log

### 2026-07-29

- Phase created. Display improvements across all views: per-column colors for name-ish columns, deficit-weighted column widths (truncated message columns get priority over padded branch columns), and configurable date/branch/author display formats via settings.json + `:settings` overlay. 207 tests pass.
