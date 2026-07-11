# git-interact (`gint`)

`gint` wraps common git operations in interactive, navigable terminal UIs. Every
view is a paginated, fuzzy-searchable list with a consistent context menu,
confirmation flows, and multi-select — so `branch`, `log`, `status`, `rebase`
and the rest all feel the same.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Cobra](https://github.com/spf13/cobra); it shells out to your system `git`, so
behavior always matches what git itself would do.

## Install

Requires **Go 1.26+** and a `git` binary on your `PATH`.

```sh
git clone <repo-url> git-interact && cd git-interact

make install    # install to $GOPATH/bin (version-stamped)
# or
make build      # build a local ./gint binary
```

`make build`/`make install` stamp the binary with the semver in [`VERSION`](VERSION)
plus the exact build commit; check it with:

```sh
gint --version   # gint 1.0.0
gint version     # gint 1.0.0 (commit abc1234) + the binary path — useful for
                  # telling a stale install on $PATH apart from a fresh build
```

## Command tour

Run any command with no arguments to open its interactive view. Aliases are in
parentheses.

| Command | Alias | What it does |
|---|---|---|
| `gint branch` | `br` | Browse branches (last commit, author, date). Checkout, delete, rename, pull/push, merge, bulk archive/delete. |
| `gint worktree` | `wt` | Browse worktrees. Checkout (prints the path — see below), create, prune, lock/unlock, delete. |
| `gint log` | `lg` | Browse commits. Checkout (detached), cherry-pick, squash, reset, merge, copy sha. |
| `gint graph` | `gr` | Commit graph (git's own layout). `-A` bases it on the current branch instead of all. |
| `gint graph-branch` | `grb` | Graph of each branch's last commit. |
| `gint rebase` | `reb` | Pick a per-commit operation (pick/reword/edit/squash/fixup/drop), then submit. Resumes an in-progress rebase and resolves conflicts inline. |
| `gint merge` | `mrg` | Merge a branch: default / ff-only / no-ff / squash. Continues or aborts an in-progress merge. |
| `gint status` | `st` | Working-tree status. Stage/unstage, diff, discard, stash, commit/amend. |
| `gint add` | | Staging operations on changed files: stage/unstage (all), restore, clean. |
| `gint stash` | `sth` | Browse stashes. Apply, pop, drop, show diff, create, clear all. |
| `gint commit` | `cm` | Commit staged changes: message, then yes / amend / stage-all / commit-&-push / no-verify. |

Every command also accepts a branch/commit name or flags for non-interactive
use, e.g. `gint branch -b feature/login`, `gint commit -I "message"`,
`gint branch <name>` (opens that branch's operation menu directly).

### `worktree` checkout

A subprocess can't change your shell's directory, so `gint worktree` checkout
hands the chosen path back to the shell. Install the wrapper once and checkout
will `cd` for you:

```sh
# bash / zsh — add to ~/.bashrc or ~/.zshrc:
eval "$(gint shell-init zsh)"
# fish — add to ~/.config/fish/config.fish:
gint shell-init fish | source
```

The wrapper runs the real binary with a temp `--cd-file`, then `cd`s to whatever
worktree you checked out; every other `gint` command passes straight through.
Without the wrapper, checkout still prints the path as its final line, so
`cd "$(gint worktree)"` works as a fallback.

## Interaction model

All interactive views share the same keys:

| Key | Action |
|---|---|
| `↑`/`k`, `↓`/`j` | move cursor (prefix with a count, e.g. `10j`) |
| `u` / `d` (or `Ctrl+U`/`Ctrl+D`) | half-page up / down |
| `←`/`h`, `→`/`l`, `PgUp`/`PgDn` | page |
| `g` / `G` | jump to top / bottom |
| `/` | fuzzy search |
| `Enter` | open the context menu for the current row |
| `Shift+X` | select mode (`space`/`x` toggle rows, Enter for bulk operations) |
| `?` | show all keys and this view's operations |
| `Esc` | close search / leave select mode / clear status |
| `q`, `Ctrl+C` | quit |

(`u`/`d` fall back to their operation binding — e.g. unstage / diff — in views
that define one.)

Common operation shortcuts (available where the operation applies):

| Key | Action |
|---|---|
| `Shift+N` | new / create |
| `Shift+C` | checkout |
| `Shift+D` | delete |
| `Shift+R` | rename |
| `Shift+M` | merge |
| `p` | pull |
| `Shift+P` | push |
| `y` | copy sha / path |

Destructive operations always confirm; the dangerous ones require a typed
phrase — e.g. `force` to force-delete a branch, `delete all` for a bulk delete,
`reset hard` for a hard reset. A shortcut runs the exact same confirmation as
its menu entry.

## Common flags

| Flag | Meaning |
|---|---|
| `-i`, `--interactive` | interactive mode (the default) |
| `-I`, `--no-interactive` | print the plain tabular list instead |
| `-S`, `--sort` | sort order |
| `-F`, `--full` | full view (complete message, absolute date, full author) |
| `-s`, `--short` | minimal view |

Because `-i` is the default, a bare `gint <command>` opens the TUI; scripts
should pass `-I` for plain output.

## Development

```sh
make build    # build ./gint (version-stamped)
make test     # go test -race ./...
make lint     # golangci-lint
make fmt      # gofmt + go vet
```

The project is built phase by phase; see [AGENTS.md](AGENTS.md) for the
architecture and context docs, and [PROMPT.md](PROMPT.md) for the full behavior
spec.
