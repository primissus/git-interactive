# git-interact (`gint`)

Build a CLI tool called **git-interact** (alias `gint`) that wraps common git operations in interactive, navigable TUI views. Command format: `gint <command> [subcommand] [options]`.

## Shared interaction model

All interactive views behave consistently:

- **Navigation**: arrow keys and vim motions (`h j k l`).
- **Pagination**: list is paginated to the terminal/viewport height (like `less`).
- **Search**: `/` starts a fuzzy search.
- **Enter**: opens a context menu of operations for the selected item.
- **`:` command palette**: opens the operation menu with a `:` prompt, autocomplete (tab), and built-in `:quit` in every view. Fuzzy-filter operations like the context menu.
- **Confirmations**: destructive operations always ask for confirmation. Dangerous ones require typed confirmation (e.g. type `force` to force-delete, `delete all` for bulk delete).
- **Select mode**: `Shift+X` enters multi-select for bulk operations.
- **Colors/markers**: current item (e.g. current branch) is highlighted with a color and a dot marker.

### Common shortcuts

| Key | Action |
|---|---|
| `/` | fuzzy search |
| `Shift+N` | new (create) |
| `Shift+C` | checkout |
| `Shift+D` | delete |
| `Shift+R` | rename |
| `p` | pull |
| `Shift+P` | push |
| `Shift+X` | select mode (bulk operations) |

Shortcuts trigger the same confirmations as the menu.

### Common flags

| Flag | Meaning |
|---|---|
| `-i`, `--interactive` | interactive mode (default) |
| `-I`, `--no-interactive` | just print the tabular list |
| `-S`, `--sort` | sort order |
| `-F`, `--full` | full view: complete commit message, date, author name |
| `-s`, `--short` | minimal view (e.g. branch name only) |

## Reference implementations

`~/src/env/src/common/git/` contains earlier Python scripts whose behavior (output format, columns, interactions) can be used as a reference for the equivalent `gint` commands:

- `git-br.py` → `branch`
- `git-lg.py` → `log`
- `git-lg-br.py` → `graph-branch`
- `git-st.py` → `status`

## Commands

### `branch` (alias `br`)

Interactive tabular list of branches. Columns: branch name, last commit, last author, relative date.

- **Sort**: last commit, created, author, off. Runtime sort cycling via `S` (Shift+S) through the four modes, with the title updating to show the current mode.
- **Filter**: by author; by date (last 1 day, 3 days, 1 week, 1 month, year to date, 1 year); merged / not-merged into the current branch; "gone" (upstream deleted — pairs well with bulk archive/delete).
- **Search**: fuzzy find with `/`.
- **Create**: an entry above the list (default focus stays on the first branch) to create a new branch.
- **Tree grouping** (`T`, or `:group`): toggles a collapsible tree view that groups branches by their `/`-separated path segments. Directories are sorted alphabetically; branches within a directory keep the active sort order. Enter on a directory row expands or collapses it. Branches show only their leaf name with progressive indentation. Disabled in `-I` mode and direct-menu mode.
- **Enter menu**: checkout, delete (confirm; force-delete requires typing `force`), pull, push, rename (pre-filled with the current branch name), copy sha (`y`), copy name (`c`), open PR (opens the branch's pull request, if any, in the browser via `gh pr view --web`).
- **Select mode** (`Shift+X`) bulk operations:
  - archive: tag as `archive/<branch-name>` and delete the branch
  - delete: confirm by typing `delete all`
  - force delete: confirm by typing `force delete`
- **Merge** (`Shift+M`): merge the selected branch into the current one, reusing the `merge` command's confirmation flow.
- **Rebase** (`B`): rebase the current branch onto the selected one via the interactive `rebase` plan view; returns to the branch list afterwards.
- **Checkout into another worktree**: checking out a branch that's already checked out somewhere else normally fails with git's raw "already used by worktree" error. Instead, `checkout` offers a prompt — "already checked out at `<path>` — move there instead?" (no / cd there) — and choosing "cd there" quits the view and hands the path to the shell exactly like `worktree`'s own checkout (see below); it needs the same `shell-init` wrapper to actually change directories. No prompt appears for a branch with no worktree of its own, or for the branch already checked out here.
- **Pull-request column**: a `pr` column (`#412 open` / `#412 draft`) fetched from `gh pr list` in the background — the list renders immediately and the column fills in once the fetch completes. Silently empty with no `gh` on PATH, no GitHub remote, no `gh auth login`, or any other failure; no flag or setting enables/disables it. Fetched once per view session, not on every refresh, so it survives a checkout or delete without re-fetching.
- **Flags**: common flags plus `-b`/`--new` (create branch), `-D`/`--delete`, `-m`/`--rename`.
- `gint branch <branchname>` (no options): shows an operations menu for that branch, expecting input; supports fuzzy matching (`pull` lowercase vs `Push` uppercase to disambiguate).

### `worktree` (alias `wt`)

Interactive list of worktrees. Columns: path (shortest form: relative, or absolute with `~`), branch, commit, relative date, pull request (same best-effort `gh`-backed column and "open PR" operation as `branch`, described there).

- **Operations**: checkout (cd into the directory), fuzzy search, sort, filter, delete, fetch (`f`), pull (`p`), push, rename branch, create worktree (`Shift+N`), prune stale worktrees, lock/unlock, copy path, open PR.
- Same shortcuts and flags as `branch`.

### `graph-branch` (alias `grb`)

Graph view of branches, showing the last commit of each only.

- **Flags**: `-A`/`--not-all` — use the current branch as base instead of all branches (all is the default).
- Interactions mirror `branch`, adapted to the graph view.

### `log` (alias `lg`)

Interactive commit list. Columns: short sha, message (truncated), relative date, author (first name + last-name initial), branches (local only by default, comma-separated), worktree dir.

- **Operations**: same as `branch`, plus:
  - **cherry-pick** (`c`): confirmation offers no / yes / no-commit and other relevant options
  - **squash**
  - **reset**: confirmation offers no / soft / mixed / hard; hard requires typing `reset hard`
  - **merge**
- Select mode supports cherry-pick on multiple commits.

### `graph` (alias `gr`)

Graph view of commits. Same `-A`/`--not-all` flag as `graph-branch`.

### `rebase` (alias `reb`)

Like interactive rebase but more direct: choose operations per commit, then a final **submit** step that asks for confirmation.

- **In-progress detection**: if invoked while a rebase is already in progress, don't start a new one — show the rebase state (branch, onto, current commit, progress e.g. 3/7) and offer: resolve conflicts (if any), continue, skip, edit the current commit, abort (with confirmation).
- **Conflicts**: when the rebase stops on a conflict, offer inline resolution per conflict: take ours / take theirs / take both / open in `$EDITOR`, then continue; abort and skip are available at any point (abort with confirmation). Always label the two sides with their branch/commit names — during a rebase, git's "ours"/"theirs" are inverted relative to a merge, so raw labels are confusing. Build this as a reusable component: the future `resolve-conflicts` command and the conflict handling in `status` should share it.

### `merge` (alias `mrg`)

Merge branches. Confirmation offers: no / yes (default merge) / fast-forward only (`--ff-only`) / no-ff (force a merge commit) / squash. When a merge is in progress: continue / abort.

### `status` (alias `st`)

Interactive status view. Reference implementation: `git-st.py` (see [Reference implementations](#reference-implementations)).

- **Enter**: toggle stage/unstage for the selected file.
- **Diff** (`d`): show the diff of the selected file.
- **Discard**: discard changes to the selected file (typed confirmation).
- **Stash**: stash the selected file.
- **Commit** (`c`): open the commit flow (see `commit`); `Shift+A` for amend.
- During conflicts: continue / abort.

### `add`

Staging operations on the working tree (list of changed files):

- **Operations**: stage, unstage, stage all, unstage all, restore file, clean (delete untracked, with confirmation).

### `stash` (alias `sth`)

Interactive list of stashes. Columns: index, message, branch, relative date.

- **Operations**: apply, pop, drop (confirm), show diff, create stash (`Shift+N`, with message input), clear all (typed confirmation: `clear all`).

### `commit` (alias `cm`)

Commit staged changes: message input, then confirm.

- **Confirmation offers**: no / yes / amend / stage all & commit / commit & push / no-verify (skip hooks) / edit (back to the message input).
- **Amend** (`Shift+A`): amend the last commit (warn if the commit is already pushed).
- **No-verify** (`Shift+V`): commit skipping hooks, also available directly as a shortcut.

## Settings (`:settings` / `:menu`)

Every interactive view opens a settings overlay via `:` → `settings` (alias `menu`). Toggled values preview live; `s` saves to `~/.config/gint/settings.json`, `esc` reverts.

- **Generic sections** (every view): appearance (system/light/dark), date format (short/long/iso), and the theme list.
- **Per-view Display sections** toggle which columns render:
  - `branch`: branch, last commit, date, author, worktree, pr.
  - `log`: sha, message, date, author, branches, worktree.
  - Hiding is TUI-only — `-I`/`--no-interactive` output always shows the full column set.
- **Format rows**: `log`'s menu carries author (short/initials/full) and branch (full/short/ultra-short) format rows; `branch`'s menu carries a worktree-path row (shortest/relative/absolute). Other views keep the original format rows.
- **Branch formats**: `full` (as-is), `short` (first rune of each leading segment, e.g. `d/d/name`), `ultra-short` (last segment with vowels stripped, e.g. `feat/auth/login-form` → `lgn-frm`).
- **Worktree paths**: `shortest` (default: relative, else `~`-abbreviated, else absolute), `relative` (relative to cwd, falling back to absolute when it escapes), `absolute`.

## Architecture

- **Language**: Go.
- **TUI**: [Bubble Tea](https://github.com/charmbracelet/bubbletea), plus its companion libraries [Bubbles](https://github.com/charmbracelet/bubbles) (ready-made components: table, list, text input, paginator) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) (styling/colors).
- **CLI / arg parsing**: [Cobra](https://github.com/spf13/cobra) — commands, subcommands, aliases, and flags. (Viper is Cobra's companion for *config files*; add it only if `gint` grows user configuration.)
- **Validation**: evaluate [go-playground/validator](https://github.com/go-playground/validator) as the default choice; search for a better fit before committing.
- **Git access**: shell out to the `git` binary (simplest and always matches user behavior); consider [go-git](https://github.com/go-git/go-git) only where parsing porcelain output gets painful.
- **Build**: a `Makefile` with at least `build`, `install`, `test`, `lint`, `fmt` targets.
- **Tests**: standard `go test`, with [teatest](https://github.com/charmbracelet/x/tree/main/exp/teatest) for TUI interaction tests.
- **Fuzzy matching**: [sahilm/fuzzy](https://github.com/sahilm/fuzzy) — powers `/` search and branch-name disambiguation.
- **GitHub CLI (`gh`)**: optional. `branch` and `worktree`'s pull-request column and "open PR" operation shell out to it exactly like git access shells out to `git`; absent, unauthenticated, or erroring, the column just stays empty — no flag, no error, no v1 hard dependency.
- **Lint**: `golangci-lint`.
- **Context docs**: generate the project context architecture with `/setup-context-architecture` (AGENTS.md / CLAUDE.md + `.context/` docs).

## Future (post-v1)

Not part of v1, but keep the design open for them:

- `remote` (alias `rem`): interactive list of remotes. Operations: add, remove, rename, set-url, fetch, prune.
- `branch-remotes` (alias `brr`): like `branch` but including remote-tracking branches. Operations: checkout (creating a local tracking branch), delete remote branch (typed confirmation), compare ahead/behind.

- `diff` (alias `df`): diff two refs (branches or commits).
  - `gint diff <ref1> <ref2>` shows the diff directly; `gint diff <ref>` diffs that ref against the working tree; `gint diff` with no args opens an interactive two-step picker (choose the base ref, then the target — branches and commits, fuzzy searchable).
  - The diff view itself is paginated and searchable like the other views: navigate by file and by hunk, collapse/expand files, `/` to search within the diff.
  - **Integration with other views**: in `branch`, `log`, `graph`, and `graph-branch`, selecting exactly two items in select mode (`Shift+X`) enables a **diff** operation between them; from a single selected item, a diff-against-current-branch (or against HEAD, in commit views) shortcut. Design select mode in v1 so this operation can plug in later.

- `resolve-conflicts` (alias `rc`): interactive resolution of merge/rebase/cherry-pick conflicts.
  - Lists conflicted files with their conflict counts; entering a file walks conflict by conflict, showing both sides labeled with their branch/commit.
  - Per conflict (or whole file): take ours / take theirs / take both / open in `$EDITOR`.
  - Header shows the operation in progress and overall progress (e.g. "rebasing `feat/x` onto `main`, 2/5 files resolved"); when everything is resolved, offer continue; abort is available at any point (with confirmation).
  - The conflict handling in `status` (continue/abort) should be designed so it can later hand off to this command.
