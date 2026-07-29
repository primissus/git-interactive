# Architecture

## In one sentence
`git-interact` (`gint`) is a Go CLI that wraps common git operations in interactive, navigable Bubble Tea TUI views (`gint <command> [subcommand] [options]`).

## Stack
- Language / runtime: Go
- TUI: Bubble Tea + Bubbles (table, list, text input, paginator) + Lip Gloss (styling)
- CLI / arg parsing: Cobra (commands, subcommands, aliases, flags)
- Fuzzy matching: sahilm/fuzzy (powers `/` search and name disambiguation)
- Git access: shell out to the `git` binary (no go-git unless porcelain parsing gets painful)
- Database: none
- External services: none

## Folder map
- `cmd/gint/` → main entry point
- `internal/cli/` → Cobra commands (one file per command; `_demo` is the hidden TUI harness)
- `internal/git/` → git exec wrapper + porcelain parsers
- `internal/tui/` → shared Bubble Tea interaction layer (list, search, menu, confirm, input, select mode, tabular renderer, `batch.go` resilient bulk ops, `?` help overlay, count-prefix/half-page nav, per-column color via `Column.Color`/`Column.Render`, theming via `themes.go` + `settings.go` + the `:settings` overlay). Commands supply `Item`s + `Operation`s to `tui.New`; they never reimplement interactions.
- `phases/p1..p8/` → development plan (PLAN.md) and status (PROGRESS.md) per phase
- `PROMPT.md` → full product spec (source of truth for behavior)

## Data flow
A Cobra command parses flags → fetches data through `internal/git` (exec `git`, parse porcelain output) → renders either an interactive Bubble Tea list/graph view (`-i`, default) or a plain tabular print (`-I`). View operations (checkout, delete, merge…) dispatch back through `internal/git`, gated by confirmation components.

## What does NOT exist (and should not be created)
- No go-git dependency by default — shell out to `git`.
- No Viper-based config system — the only config files are the small JSON ones: `~/.config/gint/keymap.json` (key/hint overrides) and `~/.config/gint/settings.json` (appearance + theme). Both load via `encoding/json`, not Viper.
- No v1 implementation of `remote`, `branch-remotes`, `diff`, `resolve-conflicts` — post-v1, but keep select mode and the conflict component open for them (PROMPT.md → Future).

## Theming & appearance
The TUI palette is theme-driven (see `.context/decisions.md` "Theming system"). `internal/tui/themes.go` registers 7 themes (default + gruvbox + solarized + catppuccin + github + nord + rose-pine), each with Light and Dark variants. `internal/tui/settings.go` persists the user's choice to `~/.config/gint/settings.json` (`appearance` ∈ {system, light, dark}; `theme` ∈ `ThemeNames()`; `dateFormat` ∈ {short, long, iso}; `branchFormat` ∈ {full, short}; `authorFormat` ∈ {short, initials, full}). At startup `cli.Execute` calls `LoadSettings` → `ApplySettings` → `setPalette` + `freezeFormats`, which freezes the resolved variant into the package-level color vars and format vars. The `:settings` (alias `:menu`) overlay (`internal/tui/settings_model.go`) previews theme/appearance/format changes live by re-running `setPalette` + `freezeFormats` + regenerating the owning `List`'s `*Styles` in place; Esc reverts, `s` saves to disk. `system` resolves once at startup via `detectOSAppearance` with terminal-background detection as a fallback.
