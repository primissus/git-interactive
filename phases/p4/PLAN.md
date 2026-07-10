# Phase 4 — `log`, `graph`, `graph-branch`

## Goal

Commit-oriented views: the interactive commit list and the two graph views.

## Scope

### `log` (`lg`)

- Columns: short sha, truncated message, relative date, author as "First L.", branches (local only by default, comma-separated), worktree dir. Reference output: `git-lg.py`.
- Operations: everything `branch` has, plus:
  - cherry-pick (`c`): confirm offers no / yes / no-commit / other relevant options.
  - squash.
  - reset: confirm offers no / soft / mixed / hard; hard requires typing `reset hard`.
  - merge (stub → real flow in phase 6).
- Select mode supports multi-commit cherry-pick.
- `-F/--full`: complete message, full date, full author name.

### `graph` (`gr`) and `graph-branch` (`grb`)

- `graph`: commit graph view. `graph-branch`: graph showing only each branch's last commit. Reference: `git-lg-br.py`.
- Flag `-A/--not-all`: base on current branch instead of all branches (all is default).
- Interactions mirror `log`/`branch`, adapted to graph rendering (selection moves through graph rows; graph glyph column is not part of fuzzy search).

## Tasks

1. Log data source (sha, message, date, author, branch decoration, worktree mapping).
2. Log view + operations; cherry-pick/squash/reset confirmation flows.
3. Multi-commit cherry-pick in select mode.
4. Graph renderer (parse `git log --graph` or render from parent data — decide; note decision in .context/decisions).
5. `graph` and `graph-branch` commands with `-A`.
6. Tests: parsers, confirmation flows, teatest navigation over a fixture history.

## Acceptance

- `gint lg` matches `git-lg.py` column behavior; graphs render correctly on a branchy fixture repo; `make lint test` clean.

## References

- PROMPT.md → `log`, `graph`, `graph-branch`; `git-lg.py`, `git-lg-br.py`.
