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

## 2026-07 · worktree "checkout" cd mechanism
- **Decision:** the `checkout` operation quits the TUI and prints the selected worktree's path as the program's final stdout line; it does not attempt to change the parent shell's directory.
- **Why:** a `gint` subprocess cannot mutate its parent shell's cwd. Printing the path is the standard pattern for this class of tool — a shell function/alias (e.g. `gintcd() { cd "$(gint worktree "$@")"; }`) can wrap it to actually `cd`. Documented here rather than shipping a shell integration script, since none exists yet.
- **Rejected:** writing the path to a fixed temp file — no benefit over stdout for this case, and adds a stale-file failure mode.
- **Status:** current

## 2026-07 · branch "created" sort heuristic
- **Decision:** the `branch` view's "created" sort uses the tip commit's author date (`%(authordate:unix)`), not committer date.
- **Why:** git has no stored branch-creation timestamp. Author date survives rebase/amend (which rewrite committer date), and matches the common `git branch --sort=-creatordate` convention.
- **Rejected:** committer date — drifts on every rebase/amend, so it doesn't track "when the branch's work started."
- **Status:** current
