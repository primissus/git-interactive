# Known issues (gotchas)

> The traps that already bit you. Each one saves an hour of the agent's time
> (and yours).

## "ours"/"theirs" are inverted during rebase
- **Happens when:** labeling conflict sides while a rebase is in progress.
- **Real cause:** during rebase, git's "ours" is the branch being rebased *onto*, the opposite of a merge.
- **Fix:** never show raw ours/theirs — the conflict component always resolves labels to branch/commit names (PROMPT.md → `rebase`).

## [PENDING: more entries as development starts]
- **Happens when:** [...]
- **Real cause:** [...]
- **Fix:** [...]

## Things that look broken but are intentional
- `-i/--interactive` being the default means bare `gint <cmd>` opens a TUI — scripts must pass `-I`.
- [PENDING]
