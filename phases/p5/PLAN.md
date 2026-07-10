# Phase 5 — `status`, `add`, `stash`

## Goal

Working-tree commands: interactive status, staging operations, and stash management.

## Scope

### `status` (`st`)

- Reference implementation: `git-st.py`.
- Enter toggles stage/unstage of the selected file.
- `d`: show diff of the selected file (paged view).
- Discard changes to the selected file (typed confirmation).
- Stash the selected file.
- `c`: open the commit flow (delegates to the `commit` command's flow — stub until phase 6, then wire the real one); `Shift+A` amend.
- During conflicts: continue / abort options. Keep the conflict-state handling behind an interface so phase 7's resolver can hook in later (PROMPT.md requires this hand-off design).

### `add`

- List of changed files with operations: stage, unstage, stage all, unstage all, restore file, clean untracked (with confirmation).

### `stash` (`sth`)

- Columns: index, message, branch, relative date.
- Operations: apply, pop, drop (confirm), show diff, create (`Shift+N`, with message input), clear all (typed `clear all`).

## Tasks

1. Status data source (`status --porcelain=v2`) with staged/unstaged/untracked/conflict states.
2. Status view: stage toggle, diff pager, discard (typed), per-file stash.
3. Conflict-state detection + continue/abort (interface for phase 7 hand-off).
4. `add` command reusing the status data source.
5. `stash` command: list, apply/pop/drop, diff, create with message, clear-all typed confirm.
6. Tests across dirty/conflicted fixture repos.

## Acceptance

- Status behavior matches `git-st.py` reference; no destructive op without confirmation; `make lint test` clean.

## References

- PROMPT.md → `status`, `add`, `stash`; `~/src/env/src/common/git/git-st.py`.
