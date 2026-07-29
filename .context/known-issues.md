# Known issues (gotchas)

> The traps that already bit you. Each one saves an hour of the agent's time
> (and yours).

## "ours"/"theirs" are inverted during rebase
- **Happens when:** labeling conflict sides while a rebase is in progress.
- **Real cause:** during rebase, git's "ours" is the branch being rebased *onto*, the opposite of a merge.
- **Fix:** never show raw ours/theirs — the conflict component always resolves labels to branch/commit names (PROMPT.md → `rebase`).

## rebase todo injection relies on a POSIX `cp` sequence editor
- **Happens when:** running `gint rebase`'s submit on a system where `GIT_SEQUENCE_EDITOR=cp -- <file>` won't run (non-POSIX shell, no `cp`).
- **Real cause:** `RunRebasePlan` writes the todo to a temp file and sets `GIT_SEQUENCE_EDITOR` to `cp -- <tempfile>`; git invokes it as `cp -- <tempfile> <todofile>` to overwrite its generated todo. This assumes a POSIX `cp`.
- **Fix:** fine on macOS/Linux (the supported targets). If Windows support is ever needed, replace the `cp` editor with a tiny gint-internal helper mode instead.

## continuing a rebase/merge must not open an editor
- **Happens when:** continuing past a `squash`/`edit`, or a merge with a message, from inside the TUI.
- **Real cause:** `git <op> --continue` opens `$EDITOR` for the (already-prepared) message by default, which blocks the alt-screen.
- **Fix:** `InProgressState.Continue` runs with `GIT_EDITOR=true`/`GIT_SEQUENCE_EDITOR=true`, so git keeps the prepared message and returns immediately.

## Things that look broken but are intentional
- `-i/--interactive` being the default means bare `gint <cmd>` opens a TUI — scripts must pass `-I`.
- `merge` ignores `-I` and always confirms (it's a one-shot wizard with no tabular form — see decisions). `-S` is attached to every command but only `branch` reorders by it; `stash` binds `p` to pop, not pull. All intentional (phase-8 audit).
- `log`'s `reset` has no keyboard shortcut on purpose: `X` is reserved framework-wide for select mode, and PROMPT.md assigns reset no key — it's menu-only.
- On a freshly `git init`'d repo (unborn branch) `log`/`graph` show an empty list rather than an error; run outside a repo, every command fails fast with "not a git repository" (via the root `PersistentPreRunE`).
- After changing the theme in `:settings`, per-column tints (e.g. SHA amber, date gray) don't update until the next `gint` invocation. Backgrounds, row styles, markers, and overlays DO update live. `Column.Color` is assigned once at command-construction time, so chasing a mid-session rebuild would add complexity for a small visual gain — see decisions.md "Theming system".
