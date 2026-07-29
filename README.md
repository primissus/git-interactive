# gint

**gint** — from **g**it-**int**eractive — wraps common git operations in
interactive, navigable terminal UIs. Every
view is a paginated, fuzzy-searchable list with a consistent context menu,
confirmation flows, and multi-select — so `branch`, `log`, `status`, `rebase`
and the rest all feel the same.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Cobra](https://github.com/spf13/cobra); it shells out to your system `git`, so
behavior always matches what git itself would do.

## Install

`gint` shells out to `git`, so a **`git` binary on your `PATH`** is the only
runtime requirement. No Go toolchain is needed to run a prebuilt binary.

### Download a prebuilt binary (recommended)

Grab the zip for your platform from the
[latest release](https://github.com/primissus/git-interactive/releases/latest):

| Platform | Asset |
| --- | --- |
| macOS (Apple Silicon) | `gint_<version>_darwin_arm64.zip` |
| macOS (Intel) | `gint_<version>_darwin_amd64.zip` |
| Linux (x86-64) | `gint_<version>_linux_amd64.zip` |
| Linux (ARM64) | `gint_<version>_linux_arm64.zip` |
| Windows (x86-64) | `gint_<version>_windows_amd64.zip` |

Unzip it, then put the `gint` binary somewhere on your `PATH` (e.g.
`/usr/local/bin`). Each zip also bundles the `LICENSE` and this `README`.
Verify the download against `checksums.txt` from the release:

```sh
shasum -a 256 -c checksums.txt   # run in the folder holding the downloaded zips
```

**macOS:** binaries are unsigned, so Gatekeeper quarantines them on first run
("cannot be opened because the developer cannot be verified"). Clear it once:

```sh
xattr -d com.apple.quarantine gint
```

### Build from source

Requires **Go 1.26+**.

```sh
git clone https://github.com/primissus/git-interactive.git gint && cd gint

make install    # install to $GOPATH/bin (version-stamped)
# or
make build      # build a local ./gint binary
```

Either way the binary is stamped with the semver in [`VERSION`](VERSION) plus the
exact build commit; check it with:

```sh
gint --version   # gint 1.0.1
gint version     # gint 1.0.1 (commit abc1234) + the binary path — useful for
                 # telling a stale install on $PATH apart from a fresh build
```

Maintainers: `make release` cross-compiles every platform above into `dist/` as
zips plus a `checksums.txt`, ready to attach to a GitHub release.

## Command tour

Run any command with no arguments to open its interactive view. Aliases are in
parentheses.

| Command | Alias | What it does |
|---|---|---|
| `gint branch` | `br` | Browse branches (last commit, author, date). Checkout, delete, rename, pull/push, merge, bulk archive/delete. |
| `gint tags` | `tag` | Browse tags (message, tagger/author, date). Checkout (detached), delete, push, bulk delete. |
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
| `u` / `d` (or `Ctrl+U`/`Ctrl+D`, `Opt+↑`/`Opt+↓`) | half-page up / down |
| `←`/`h`, `→`/`l`, `PgUp`/`PgDn` | page |
| `g` / `G` | jump to top / bottom — prefix `g` with a row number (from the gutter) to jump there, e.g. `12g` |
| `/` | fuzzy search |
| `:` | command palette (type `settings` or `menu` to open the [settings overlay](#theming)) |
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

## Configuration

### Theming

`gint` ships with seven color themes — `default`, `gruvbox`, `solarized`,
`catppuccin`, `github`, `nord`, and `rose-pine` — each with Light and Dark
variants. Hold `:` to open the command palette, type `settings` (or its
alias `menu`), press Enter, and a settings overlay opens with an appearance
toggle (System / Light / Dark) and a theme list with live swatch previews:

```
╭─ Settings ──────────────────────────────╮
│  Appearance                             │
│  ◉ System   ○ Light   ○ Dark            │
│                                         │
│  Theme                                  │
│  ▸ default          ■  ■  ■  ■          │
│    gruvbox          ■  ■  ■  ■          │
│    solarized        ■  ■  ■  ■          │
│    ...                                  │
│                                         │
│  ↑/↓ select · ←/→ toggle · enter select │
│  · s save · esc cancel                  │
╰─────────────────────────────────────────╯
```

Navigation: `j`/`k` between rows, `←`/`→` to cycle the appearance option
(on the appearance row), `Enter` to preview a theme, `s` to save to
`~/.config/gint/settings.json` and close, `Esc` to revert and close. Changes
preview live — row backgrounds, highlights, and the overlay update immediately;
per-column tints (e.g. the SHA amber) update on the next invocation.

`System` (the default when no settings.json exists) detects the OS dark-mode
preference once at startup (macOS `AppleInterfaceStyle`, GNOME `gsettings`,
Windows `reg query`, with terminal-background detection as a fallback) and
freezes the resolved variant for the session — toggling OS dark mode while
`gint` is running doesn't affect the current session; restart to pick it up.

The settings file is plain JSON:

```json
{
  "appearance": "system",
  "theme": "gruvbox"
}
```

Valid values: `appearance` ∈ `{system, light, dark}` (empty = `system`);
`theme` ∈ `ThemeNames()` (empty or unknown = `default`). A missing or malformed
file is not fatal — `gint` warns and falls back to defaults, exactly like
`keymap.json`.

### Keymap overrides

Every displayed hint — an operation's shortcut key and label, and the generic
footer/prompt/help text — has a built-in default and can be overridden without
rebuilding, via `~/.config/gint/keymap.json` (or `$XDG_CONFIG_HOME/gint/keymap.json`
if set). The file is entirely optional and only needs to list what you want to
change; everything else keeps its default. A missing or malformed file is not
fatal — a parse error is printed as a warning and gint falls back to defaults.

```json
{
  "operations": {
    "status.toggle stage": { "key": "a" },
    "branch.checkout": { "key": "o", "label": "check out" }
  },
  "chrome": {
    "footer": "j/k move · / search · enter menu · q quit"
  }
}
```

`operations` keys are `"<command>.<operation>"` — the command is the name
shown in the [command tour](#command-tour) table (`status`, `branch`, `log`,
`tags`, `rebase`, `stash`, `worktree`, `graph`, `add`, or `conflict` for the
shared take-ours/take-theirs/take-both/edit/continue/skip/abort conflict-
resolution keys used by both `status` and `rebase`); the operation is its name
as shown in that view's `?` help overlay. `chrome` fields cover the generic
list/menu/confirm/input/batch chrome; see `internal/tui/keymap.go`'s `Chrome`
struct for the full field list.

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
