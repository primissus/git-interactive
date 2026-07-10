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
- `internal/tui/` → shared Bubble Tea interaction layer (list, search, menu, confirm, input, select mode, tabular renderer, `batch.go` resilient bulk ops, `?` help overlay, count-prefix/half-page nav, per-column color via `Column.Color`/`Column.Render`). Commands supply `Item`s + `Operation`s to `tui.New`; they never reimplement interactions.
- `phases/p1..p8/` → development plan (PLAN.md) and status (PROGRESS.md) per phase
- `PROMPT.md` → full product spec (source of truth for behavior)

## Data flow
A Cobra command parses flags → fetches data through `internal/git` (exec `git`, parse porcelain output) → renders either an interactive Bubble Tea list/graph view (`-i`, default) or a plain tabular print (`-I`). View operations (checkout, delete, merge…) dispatch back through `internal/git`, gated by confirmation components.

## What does NOT exist (and should not be created)
- No go-git dependency by default — shell out to `git`.
- No config-file system (Viper) unless user configuration actually appears.
- No v1 implementation of `remote`, `branch-remotes`, `diff`, `resolve-conflicts` — post-v1, but keep select mode and the conflict component open for them (PROMPT.md → Future).
