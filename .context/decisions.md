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

## 2026-07 · Graph rendering approach
- **Decision:** parse `git log --graph --pretty=format:...`, using a sentinel control character (`\x02`) as the field-format's first byte so the ASCII graph glyphs (which git prepends to the format string on every commit line, and are the entire content of pure connector lines) can be split from the commit data by string index rather than re-deriving the graph from parent/child edges.
- **Why:** git's own graph layout algorithm is exactly what users expect and is nontrivial to reimplement (merge/branch column packing); shelling out for it and parsing the text is far simpler than rendering from `Commit.Refs`/parent data ourselves.
- **Rejected:** rendering the graph from parent-commit data in Go — would have to reimplement git's column-packing algorithm for no behavioral benefit.
- **Status:** current

## 2026-07 · `log`/`graph` operations scope
- **Decision:** `log` and `graph` offer checkout (detached), copy-sha, cherry-pick, squash, reset, and a merge-into-current stub — not branch's delete/rename/pull/push, despite PLAN.md phrasing operations as "everything branch has, plus…".
- **Why:** delete/rename/pull/push don't have a coherent meaning applied to a commit (as opposed to a branch); PROMPT.md's own operation list for `log` never actually lists them, only cherry-pick/squash/reset/merge as the commit-specific additions.
- **Rejected:** implementing literal branch-shaped delete/rename/pull/push against a commit's SHA — no sensible git operation backs them.
- **Status:** current

## 2026-07 · `squash` scope (phase 4)
- **Decision:** `log`'s squash operation only squashes HEAD into its immediate parent (soft reset one commit back + amend); attempting it on any other commit returns a status message pointing at phase 7.
- **Why:** squashing an arbitrary non-HEAD commit into its parent while preserving every commit above it requires rebase machinery, which is explicitly phase 7's scope (PROMPT.md → `rebase`). Building a one-off partial rebase here would duplicate that work non-reusably.
- **Rejected:** a general-purpose squash-any-commit implementation now — belongs in phase 7's reusable rebase engine.
- **Status:** current

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
