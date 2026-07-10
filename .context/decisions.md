# Decisions made

> One entry per decision. What matters is the "why" and what was rejected.

## 2026-07 · TUI stack: Bubble Tea ecosystem
- **Decision:** Bubble Tea + Bubbles + Lip Gloss for all interactive views.
- **Why:** mature Go TUI ecosystem with ready-made components (table, list, text input, paginator) and a testing story (teatest).
- **Rejected:** [PENDING: alternatives considered, e.g. tview/gocui]
- **Status:** current

## 2026-07 · CLI framework: Cobra
- **Decision:** Cobra for commands, subcommands, aliases, flags.
- **Why:** standard for multi-command Go CLIs.
- **Rejected:** Viper for config — deferred until `gint` actually grows user configuration.
- **Status:** current

## 2026-07 · Git access: shell out to `git`
- **Decision:** exec the `git` binary; parse porcelain output.
- **Why:** simplest and always matches user behavior.
- **Rejected:** go-git as the default — reconsider only where porcelain parsing gets painful.
- **Status:** current

## 2026-07 · Validation library
- **Decision:** [PENDING: evaluate go-playground/validator vs lighter options — scheduled for phase 8]
- **Why:** —
- **Rejected:** —
- **Status:** open

## [Date] · Graph rendering approach
- **Decision:** [PENDING: parse `git log --graph` vs render from parent data — decide in phase 4]
- **Status:** open
