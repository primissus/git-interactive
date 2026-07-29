# Phase 13 — Display improvements: per-column colors, deficit-weighted widths, configurable formats

## Goal

Add per-column color tints to every column type, give truncated flex columns (message) priority over padded ones (branch name) for leftover terminal width, and make date/branch/author display formats configurable via `settings.json` + the `:settings` overlay with live preview.

## Decisions

- **Per-column colors**: sha/date/author/refs already tinted; add `ColorName` to primary identifier columns (branch name, tag name, worktree path/branch, stash branch). Message columns stay default-foreground (they're the reading content). See `.context/decisions.md` for the full list.
- **Deficit-weighted width**: the existing `layout()` computes natural widths capped at `MaxWidth`, then dumps all leftover width into flex columns evenly. The new algorithm grows flex columns that are *truncated* (width < natural) first, one cell at a time round-robin, up to their natural width; remaining slack splits evenly. This means short branch names → message column expands past MaxWidth up to full content width.
- **Settings format**: `dateFormat` ∈ {short, long, iso}; `branchFormat` ∈ {full, short}; `authorFormat` ∈ {short, initials, full}. Defaults: short/full/short. Normalized on load, frozen into package vars by `ApplySettings` — same pattern as theme/appearance.
- **`-F` flag wins**: log/graph's `-F/--full` still shows absolute ISO date + full author, overriding settings.
- **`:settings` overlay**: three toggle rows between appearance and theme list. Live preview works for free — cells re-render from package vars every frame.

## Scope

### `tui/format.go` (new) — format helpers + frozen vars

- `ShortDate(unix, now)`: `<60s`→`now`, `<60m`→`N min`, `<24h`→`N hr`, `<30d`→`N day`, `<365d`→`N mth`, else `N yr`
- `FormatDate(unix, rel)`: dispatches per `activeDateFormat`
- `ShortBranch(name)`: all but last `/`-segment → first rune (e.g. `d/d/name`)
- `FormatBranch(name)`: dispatches per `activeBranchFormat`
- `ShortAuthor(name)`: first name + last initial + "." (e.g. `Test U.`)
- `InitialsAuthor(name)`: first rune of each word uppercased (e.g. `TU`)
- `FormatAuthor(name)`: dispatches per `activeAuthorFormat`
- Package vars: `activeDateFormat`, `activeBranchFormat`, `activeAuthorFormat`
- `freezeFormats`, `snapshotFormats`, `nowFunc` (for test stubbing)

### `tui/settings.go` — new Settings fields

- `DateFormat`, `BranchFormat`, `AuthorFormat` added to `Settings` struct (JSON `omitempty`)
- `LoadSettings`: normalize each to known values, fallback to defaults
- `ApplySettings`: also calls `freezeFormats`
- `CurrentSettings`: snapshots format vars

### `tui/settings_model.go` — overlay toggle rows

- Three rows: Date (short/long/iso), Branch (full/short), Author (short/initials/full)
- Cursor: 0=appearance, 1=date, 2=branch, 3=author, 4+=themes
- ←/→/enter cycle options; preview updates format vars live; Esc reverts

### `tui/keymap.go` — chrome strings

- Added `SettingsDateFormat`, `SettingsBranchFormat`, `SettingsAuthorFormat` to `Chrome` + `chromeOverride` + `applyChromeOverride` + defaults

### `tui/render.go` — deficit-weighted layout

- `layout()` now computes natural (uncapped) widths alongside capped widths
- Leftover grows truncated flex columns first (round-robin by deficit, up to natural); remainder split evenly among all flex columns (existing fallback)

### Git data plumbing

- `git/log.go`: `%ct` → `Commit.CommitUnix`
- `git/stash.go`: `%ct` → `Stash.Unix`
- `cli/worktree.go`: `commitRelDate` returns `(rel, unix, error)`

### Column definition updates

| File | Changes |
|---|---|
| `cli/branch.go` | branch += `ColorName`; last commit += `Flex:true`; date MinWidth 7 |
| `cli/tags.go` | tag += `ColorName`; message += `Flex:true`; date MinWidth 7 |
| `cli/stash.go` | branch += `ColorName`; date MinWidth 7 |
| `cli/worktree.go` | path, branch += `ColorName`; date MinWidth 7 |
| `cli/log.go` | date MinWidth 7 |
| `cli/graph.go` | date MinWidth 7 |

### Item cell formatting

- `branchItem`, `logItem`, `graphItem`, `tagItem`, `stashItem`, `worktreeItem`, `rebaseSelectItem`: use `FormatBranch`, `FormatDate`, `FormatAuthor` in their `Columns()` methods

### Tests

- `tui/format_test.go` — ShortDate buckets, ShortBranch, ShortAuthor, InitialsAuthor, Format* dispatch, freezeFormats, edge cases
- `tui/settings_test.go` already covers normalization; pass on new fields
- `tui/list_test.go` — settings overlay test updated for the 3 new cursor rows
- `cli/log_test.go` — authorInitial → tui.ShortAuthor

### Docs

- `phases/README.md` — add p13 row
- `.context/decisions.md` — this entry
- `.context/architecture.md` — settings.json description updated

## Files touched

`internal/tui/format.go` (new), `internal/tui/format_test.go` (new), `internal/tui/settings.go`, `internal/tui/settings_model.go`, `internal/tui/render.go`, `internal/tui/keymap.go`, `internal/git/log.go`, `internal/git/stash.go`, `internal/cli/{branch,log,graph,tags,stash,worktree,rebase_select}.go`, `internal/cli/log_test.go`, `internal/tui/list_test.go`, `phases/p13/{PLAN,PROGRESS}.md`, `.context/decisions.md`, `.context/architecture.md`, `phases/README.md`.
