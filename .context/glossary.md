# Glossary & entities

## Domain terms
- **Shared interaction model** → the common behavior contract of all views (navigation, `/` fuzzy search, Enter → context menu, confirmations, select mode, highlight markers). Defined in PROMPT.md.
- **Typed confirmation** → dangerous operations require typing an exact phrase (`force`, `delete all`, `force delete`, `reset hard`, `clear all`).
- **Select mode** → multi-select (`Shift+X`) enabling bulk operations; designed to later host a two-item `diff` operation.
- **Archive (branch)** → tag a branch as `archive/<branch-name>` and delete it.
- **Gone branch** → local branch whose upstream was deleted; a `branch` filter, pairs with bulk archive/delete.
- **Direct-menu mode** → `gint branch <name>`: skip the list, open the operations menu for that branch with fuzzy op matching.
- **Conflict-resolution component** → reusable ours/theirs/both/$EDITOR walk-through shared by `rebase`, `status`, and the future `resolve-conflicts`; always labels sides with branch/commit names.
- **Resilient bulk (BatchSpec)** → a destructive bulk op that processes targets one at a time, pausing on each failure to ask continue/stop/all, then reports a `deleted N · failed M` summary. `branch` bulk deletes run oldest-commit-first.
- **Motion count prefix** → typing digits before a motion repeats it (`10j` = down ten rows).
- **Help overlay (`?`)** → lists the shared nav keys plus the current view's operation shortcuts.
- **Column tint** → `Column.Color`/`Column.Render` give each column its own hue on ordinary rows; graph lanes are colored by column position. Suppressed on the cursor/current row and in `-I` output.
- **shell-init wrapper** → `gint shell-init [bash|zsh|fish]` prints a `gint()` function that makes `worktree` checkout `cd` the calling shell via a temp `--cd-file`.
- **Command palette (`:`)** → shared framework feature: opens the operation menu with a `:` prompt, tab autocomplete, and a built-in `:quit`. Fuzzy-filter operations like the context menu. Present in every view.
- **Branch grouping (tree mode)** → `T` toggles a collapsible tree view of branches split on `/`. Directories are sorted alphabetically; branches within keep the active sort order. Enter on a directory row expands/collapses it. Branches show only their leaf name with progressive indentation.
- **Dynamic rename pre-fill** → `InputSpec.InitialFrom` resolves an initial text value from the operation's target items at invocation time, so rename (`R`) pre-fills the input with the current branch name.

## Main entities
- **Command views** → `branch`(br), `worktree`(wt), `graph-branch`(grb), `log`(lg), `graph`(gr), `rebase`(reb), `merge`(mrg), `status`(st), `add`, `stash`(sth), `commit`(cm).
- **Operation** → a menu entry + optional shortcut on a selected item; shortcuts trigger the same confirmations as the menu.

## Acronyms & internal names
- **gint** → the binary/alias for git-interact.
- **`-A/--not-all`** → graph views: base on current branch instead of all branches.
- **Reference scripts** → earlier Python implementations in `~/src/env/src/common/git/` (`git-br.py`, `git-lg.py`, `git-lg-br.py`, `git-st.py`) defining expected output formats.
