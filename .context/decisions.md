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

## 2026-07 · Validation library — internal `validate`, not go-playground/validator (phase 8)
- **Decision:** a small in-house `internal/validate` package (plain functions, no struct tags) validates TUI text inputs. `BranchName` applies git's own ref-format rules (the subset `git check-ref-format --branch` enforces) in-process; commit/stash messages need only the framework's built-in non-empty guard. Inputs opt in via `InputSpec.Validate func(string) error`; optional inputs opt out of the non-empty guard via `InputSpec.AllowEmpty`.
- **Why:** go-playground/validator is a reflection/struct-tag validator built for web/API DTOs (email, URL, numeric ranges). Git ref validity is not expressible as a tag — it would require a custom validator function anyway, at which point the dependency adds only reflection overhead and a heavier import graph. A tiny function package is a better fit: zero new deps, unit-testable without a repo, and it yields a precise per-rule message the TUI shows inline under the field.
- **Rejected:** go-playground/validator (wrong shape, dead weight); shelling out to `git check-ref-format` (needs a repo/subprocess per keystroke-batch and gives a terse generic error rather than a named rule).
- **Status:** current

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

## 2026-07 · commit/merge as standalone Flow commands
- **Decision:** `commit` and `merge` are the first commands with no list to select from, so they get a new shared `internal/tui.Flow` component (message input → multi-choice confirm, with an "edit" choice that loops back to the input) instead of `internal/tui.List`. Wiring the same choice logic into `status`'s inline commit/amend and `branch`/`log`/`graph`'s merge-into-current reuses `List`'s existing Input+Confirm operation chaining — no second List-hosted flow component was needed for those.
- **Why:** `List`'s Input+Confirm chaining already covers "message then choice" for operations *inside* an existing list; `commit`/`merge` invoked standalone have no rows to hang an Operation off, so they need their own minimal `tea.Model`. Nesting a second `tea.Program` inside a running `List`'s `tea.Program` isn't supported by Bubble Tea (unlike `tea.ExecProcess`, which hands off to an *external* process, not another `tea.Model`), which is why status's commit/amend do NOT launch a `Flow` — they run inline as ordinary `List` operations instead.
- **Rejected:** giving status's commit/amend a nested `Flow` (not supported without releasing/reacquiring the terminal manually) and giving `commit`/`merge` their own hand-rolled input/confirm loop duplicating `List`'s already-built `inputModel`/`confirmModel`.
- **Status:** current

## 2026-07 · merge has no -I/non-interactive mode
- **Decision:** `gint merge` always launches its confirm Flow; it does not support `-I/--no-interactive` the way list-backed commands do.
- **Why:** merge is a one-shot confirmation wizard, not a list with a tabular alternative — there's no "table of rows" for `-I` to print. `commit -I <message>` remains supported since a plain commit has an unambiguous non-interactive form (message + implicit "yes"); merge's mode choice (default/ff-only/no-ff/squash) doesn't have an equally obvious default to assume non-interactively.
- **Rejected:** defaulting `-I` to `MergeDefault` silently — a merge is exactly the kind of operation PROMPT.md's shared model says should always confirm; skipping that under `-I` would violate the "destructive operations always confirm" rule for a command that can produce conflicts.
- **Status:** current

## 2026-07 · status's "Enter toggles stage" vs. the shared List's Enter
- **Decision:** `status`'s stage/unstage toggle is bound to the `t` key and listed first in the item context menu, not bound directly to Enter.
- **Why:** PROMPT.md specifies "Enter: toggle stage/unstage for the selected file", but phase 2's shared `internal/tui.List` always uses Enter to open the context menu — every other view (and the framework's own key-handling switch) depends on that invariant. Special-casing Enter for one view would break the "commands never reimplement interactions" rule from `.context/conventions.md`.
- **Rejected:** overriding Enter's meaning per-view — would require plumbing a List-level "raw Enter" escape hatch that no other command needs, for a single-view convenience.
- **Status:** current

## 2026-07 · conflict-state interface (status ↔ future resolve-conflicts)
- **Decision:** `internal/git.InProgressState` (detection via `DetectInProgress`, plus `Continue`/`Abort`) is the single conflict-state primitive; `status` (phase 5) only offers continue/abort, but the type carries no status-specific assumptions.
- **Why:** PROMPT.md requires phase 5's conflict handling and the future `resolve-conflicts` command (Future section) to share their conflict-state design. Keeping detection and continue/abort in `internal/git`, independent of any TUI, means `resolve-conflicts` can reuse `DetectInProgress` directly and layer per-conflict resolution on top without status's code changing.
- **Rejected:** detecting conflict state ad hoc inside `internal/cli/status.go` — would leave phase 7 re-deriving the same `.git`-directory marker-file logic.
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
- **Status:** superseded by "worktree checkout cd via shell-init + --cd-file" below.

## 2026-07 · worktree checkout cd via `shell-init` + `--cd-file` (post-v1 feedback)
- **Decision:** ship a `gint shell-init [bash|zsh|fish]` command that prints a `gint()` wrapper function; the wrapper runs the real binary with a temp `--cd-file` and `cd`s to the path checkout writes there. The hidden `--cd-file` flag on `worktree` takes the checkout path instead of stdout when set; bare-stdout printing remains the fallback for users without the wrapper.
- **Why:** the print-to-stdout mechanism is unreliable in practice — the interactive view runs under `tea.WithAltScreen`, whose escape sequences also go to stdout, so `cd "$(gint worktree)"` captures terminal noise, not just the path. Handing the path off through a caller-provided file sidesteps stdout entirely and makes the common case (`eval "$(gint shell-init zsh)"`) just work. `shell-init` is exempt from the repo-check `PersistentPreRunE` since it's evaluated from shell rc files outside any repo.
- **Rejected:** a *fixed* temp file (stale-file races, the original rejection) — the wrapper mints a fresh `mktemp` file per call; rendering the TUI to `/dev/tty`/stderr so stdout stays clean (larger change, and the file handoff is more robust across shells).
- **Status:** current

## 2026-07 · resilient bulk delete — per-failure prompt, oldest-first (post-v1 feedback)
- **Decision:** destructive bulk operations run through `tui.BatchSpec` (set on `Operation.Batch` instead of `Run`) rather than a straight-line loop. The `List` processes targets one at a time; a failing item pauses in `modeBatchPrompt` asking continue/stop/all (`y`/`n`/`a`), then finishes with a `deleted N · failed M (reasons)` summary. `branch` delete/force-delete/archive order targets oldest-commit-first via an `oldestFirst` comparator on `CommitUnix`.
- **Why:** the previous loops returned on the first error, stranding the rest of a bulk delete with no recourse. Peeling stale branches oldest-first and surviving per-item failures is what the user expects from a bulk cleanup. Steps run synchronously inside `Update` (like `Run`), pausing only to render the prompt, so no goroutine/message plumbing is needed.
- **Rejected:** plow-through-and-summarize with no prompt (loses the user's per-failure control they asked for); collecting failures silently (the old loop's abort was already too quiet).
- **Status:** current

## 2026-07 · shared-view polish: highlight, per-column color, nav, help (post-v1 feedback)
- **Decision:** several shared-`List` refinements: (1) the current/cursor row's marker and select checkbox render **unstyled** on the highlighted row so the row background spans the whole line — a pre-styled glyph's SGR reset was cutting the highlight short; (2) `Column.Color`/`Column.Render` tint each column on ordinary rows (suppressed on the cursor/current row), with `ColorizeGraphPrefix` coloring graph lanes by column position; (3) navigation gains a numeric count prefix (`10j`), `u`/`d` (and `Ctrl+U`/`Ctrl+D`) half-page jumps, and a `?` help overlay listing nav + this view's operations; (4) `x` toggles selection in select mode; (5) the bulk menu now also offers `ScopeList` operations so select-mode views (e.g. `add`) surface stage-all / clean / restore, not just item-bulk ops.
- **Why:** direct user feedback on v1. `u`/`d` yield to a view's own binding when one exists (unstage in `add`, diff in `status`/`stash`), so half-page nav is additive, not a regression. Per-column `Render` must preserve display width (SGR-wrap only) because the table renderer measures with `runewidth`; the non-interactive `-I` renderer stays plain.
- **Rejected:** global `u`/`d` that shadow view operations; embedding ANSI in cell content before width measurement (breaks alignment); a separate help program (an overlay reuses the existing footer slot).
- **Status:** current

## 2026-07 · rebase driven by a generated todo, not `-i` in an editor (phase 7)
- **Decision:** `rebase` builds the interactive-rebase todo itself and feeds it to `git rebase -i` via `GIT_SEQUENCE_EDITOR` (set to `cp -- <tempfile>`, which overwrites git's todo), with `GIT_EDITOR=true` so message-editing steps never open an editor. `reword` is expressed as `pick` + `exec git commit --amend -m <msg>` so the new message is captured in the planning view up front, not through a blocking editor at replay time.
- **Why:** the whole point is to run the rebase from inside a Bubble Tea TUI, where launching the user's `$EDITOR` on git's todo (or on a squash/reword message) would fight the alt-screen and block. Generating the todo and neutralizing the editor keeps the entire rebase non-interactive from git's side; the TUI owns all the interaction.
- **Rejected:** launching `git rebase -i` with the real editor (breaks inside the TUI); using `reword`/`squash` and driving git's message editor with a per-commit `GIT_EDITOR` helper script (fragile — has to map each editor invocation back to the right commit's message). `exec … --amend` sidesteps the message editor entirely.
- **Status:** current

## 2026-07 · rebase base selection (phase 7)
- **Decision:** `PlanRebase` offers the commits ahead of the upstream fork-point (`merge-base HEAD @{upstream}`) when the branch has an upstream; otherwise it offers the most recent commits capped at 20, rebased onto the parent of the oldest (or `--root` when the oldest is a root commit).
- **Why:** the upstream fork-point is the natural "your commits" set for the common feature-branch case. Without an upstream there's no principled fork-point, so a bounded recent window keeps the list useful and fast rather than listing all of history.
- **Rejected:** always listing full history (slow, overwhelming) and requiring an explicit base argument (PROMPT.md frames rebase as per-commit operations on the current branch, not "rebase onto X").
- **Status:** current

## 2026-07 · one conflict component, composed not nested (phase 7)
- **Decision:** the conflict-resolution component is a set of `tui.Operation`s (`fileResolutionOps` = take ours/theirs/both/$EDITOR, plus `continueSkipAbortOps`) parameterized by a `pathOf` row-extractor. `rebase` hosts them in a dedicated resolver `List` (`runConflictResolver`); `status` appends `fileResolutionOps` to its own list. Side labels come from `git.ResolveSides`, which inverts ours/theirs for rebase.
- **Why:** PROMPT.md requires `rebase`, `status`, and the future `resolve-conflicts` to share one component. Since a `tea.Program` can't be nested inside a running `List` (see the commit/merge decision), the shareable unit is the operation set, not a whole sub-program — each host composes it into its own list. Resolving labels in `internal/git` keeps the inversion rule in one place.
- **Rejected:** a standalone resolver program that `status` launches nested (unsupported); duplicating take-ours/theirs/both logic in status and rebase; showing raw "ours"/"theirs" (forbidden — inverted under rebase, see known-issues).
- **Status:** current

## 2026-07 · branch "created" sort heuristic
- **Decision:** the `branch` view's "created" sort uses the tip commit's author date (`%(authordate:unix)`), not committer date.
- **Why:** git has no stored branch-creation timestamp. Author date survives rebase/amend (which rewrite committer date), and matches the common `git branch --sort=-creatordate` convention.
- **Rejected:** committer date — drifts on every rebase/amend, so it doesn't track "when the branch's work started."
- **Status:** current

## 2026-07 · select-mode diff hook is the SelectedItems + bulk-op contract (phase 8)
- **Decision:** the future `diff`-between-two-items integration (PROMPT.md → Future → diff, "design select mode in v1 so this operation can plug in later") has no v1 implementation; the reserved hook is `List.SelectedItems()` (returns the selection in list order) plus the bulk-operation registry. When `diff` lands it registers a `Bulk`/`BulkOnly` op that acts when `len(SelectedItems()) == 2`, exactly like `log`'s bulk cherry-pick. No placeholder op is added in v1.
- **Why:** the framework already exposes the selection and routes bulk ops through the same menu; adding a real hook now is design-only future-proofing, and a dead "diff" menu entry that errors would be worse UX than none. `graph`/`graph-branch` register no bulk ops in v1, so their select mode reports "no operations available" until `diff` (or another bulk op) is added — accepted, since v1 gives them no bulk action.
- **Rejected:** shipping a stub diff op that returns "not implemented" (misleading menu entry); a bespoke two-item callback separate from the bulk-op path (would fork the menu routing the diff op will reuse).
- **Status:** current

## 2026-07 · phase-8 consistency audit — accepted divergences
- **Decision:** two low-severity key/flag divergences from the shared model are kept as-is: (1) `stash` binds `p` to **pop** (the common table reserves `p` for pull, but stashes have no pull, and `p`=pop is the natural mnemonic); (2) `-S/--sort` is attached to every command for a uniform flag surface but only `branch` reorders by it — `worktree` always sorts by path and the commit views pass it as a display label only.
- **Why:** the audit's one real bug (a dead `reset` shortcut bound to the reserved `X` in `log`) and the `graph -I` missing-marker cosmetic were fixed; these two remaining items are intentional, harmless, and churning them (remapping the obvious pop key, or building per-view sort parity) would cost more than the consistency gained. Recorded so a future audit doesn't re-flag them.
- **Rejected:** remapping stash `p`; making `-S` a hard error on commands that ignore it; implementing full worktree/commit sort parity in v1.
- **Status:** current

## 2026-07 · branch view ergonomics (p11)

- **`G` kept as jump-to-bottom everywhere** — the tree-grouping key is `T` (tree), also reachable via `:group`. No per-view override needed.
- **Sort cycles, no chooser** — `S` (Shift+S) advances through sort modes one at a time, showing the current mode in the title and a status line. This is faster than a chooser menu and more vim-like.
- **Recursive collapsible tree** — `/` splits build a prefix tree; directories sort alphabetically first, then branches in the active sort order. Branches show only the leaf segment with progressive indentation (2 spaces/level). Groups implement `DefaultActioner` → enter toggles collapse. No framework changes needed — groups are normal items with a `toggle group` op.
- **`:` command palette is a shared framework feature** — opens the operation menu (item or bulk) with a `:` prompt, tab autocomplete to the top fuzzy match, and a built-in `quit` entry. Aligns with p10 task 2 design (same `Chrome.CommandPrompt`, same menu-open mechanic). The palette is built on top of the existing `menuModel` with a `command bool` flag.
- **Rename pre-fill via `InitialFrom`** — `InputSpec` gains `InitialFrom func([]Item) string`, resolved at invocation time so the rename op can read the current branch name. Static `Initial` is unchanged; `InitialFrom` wins when both are set.
- **`c` copies branch name** — reuses the existing `copyToClipboard` stub pattern from `y` (copy sha). No framework change.
- **`sortLabelMsg` / `SetSortLabel`** — small framework addition so commands can update the title's `sort:` hint at runtime, composed by the command (e.g. `sort <mode>` or `sort <mode> · tree`).
- **Why:** the user wanted vim-like ergonomics — `:` for commands, cycle-sort, tree grouping — plus pre-filled rename. Each feature reuses existing framework patterns (ops, DefaultActioner, InputSpec) so surface area is minimal.

## 2026-07 · Theming system: system/light/dark + 7 color themes (`:settings`/`:menu`)

- **Decision:** gint gains a theming layer with three appearance states (`system`, `light`, `dark`) and seven registered themes (`default`, `gruvbox`, `solarized`, `catppuccin`, `github`, `nord`, `rose-pine`), each carrying Light + Dark color variants. The user's choice persists to `~/.config/gint/settings.json` (a sibling of `keymap.json`); a new `:settings` overlay (aliased `:menu`, opened from the existing `:` command palette) toggles appearance and selects a theme with live color-swatch previews. Changes preview live (the overlay re-runs `setPalette` and regenerates the owning `List`'s `*Styles` in place), Esc reverts to the pre-overlay palette, `s` saves to disk and reports via the footer Status line.
- **Why:** user request — smooth light/dark experience across developer-preferred palettes. `system` (the first-run default with no settings.json) preserves the prior adaptive behavior: `resolveAppearance` shells out to `defaults read` (macOS), `gsettings` (GNOME), `reg query` (Windows), falling back to `lipgloss.HasDarkBackground()` terminal detection, then `dark`. Resolving once at startup and freezing via plain `lipgloss.Color` (not `AdaptiveColor`) avoids per-render re-detection and is consistent with peers (lazygit, gh-dash). Keeping `default` as a registered theme means users can always roll back.
- **Rejected:**
  - Removing `AdaptiveColor` entirely — breaks the no-config out-of-box experience for users whose terminal advertises a background.
  - Freezing to a single appearance without `system` — users on macOS automatic dark-mode switching would have to re-edit settings.json manually on every OS toggle.
  - Per-render re-resolving of `system` — measurable cost and no UX benefit (live OS toggles while gint is running can't be applied to the running TUI without a restart anyway).
  - Updating `Column.Color` tints live — they're assigned at command construction time; chasing a rebuild would add complexity for a tiny visual gain (row backgrounds + markers + overlays all update live; column-specific tints catch up on the next invocation). Tracked as a known-issue tradeoff, not a defect.
  - Adding `S` as a shortcut for settings — collides with the existing `branch` sort-cycle binding; `:settings` via the palette is enough.
  - Two separate "gruvbox light" / "gruvbox dark" theme entries — user confirmed a single `gruvbox` theme with Light = light soft and Dark = medium dark matches the requested `depending on selected appearance` semantics.
- **Status:** current

## 2026-07 · Display improvements: per-column colors, deficit-weighted widths, configurable formats (p13)

### Per-column colors
- **Decision:** every primary-identifier column (branch name, tag name, worktree path, worktree branch, stash branch) gets `tui.ColorName` tint. Sha/date/author/refs already have their tints. Message and status-code columns stay default-foreground — they're the reading content, and coloring everything hurts scannability.
- **Why:** the existing `Column.Color` infra (added in p8) was only applied to sha/date/author/refs. Filling the gaps makes tables more readable without adding new infrastructure.
- **Rejected:** coloring message columns (distracting); adding new color slots (the 5 existing tints are enough to distinguish all column types).

### Deficit-weighted column widths
- **Decision:** `List.layout()` computes natural (uncapped) widths alongside capped ones. Leftover terminal width grows flex columns that are *truncated* (width < natural) first, one cell at a time round-robin, up to their natural width; remaining slack is split evenly among all flex columns (existing fallback).
- **Why:** with the old even-split algorithm, short branch names → all leftover padding goes to the branch (flex) column while the message column stays stuck at MaxWidth. Users want message columns to absorb available space when branch names are short. Making message columns `Flex: true` + deficit weighting achieves this naturally.
- **Rejected:** capping flex growth at MaxWidth (defeats the purpose); making non-flex columns growable (would break the contract that MaxWidth is a hard cap); a more complex proportional-deficit formula (round-robin is simple and fast — ≤400 iterations per frame).

### Short date format buckets
- **Decision:** `ShortDate(unix, now)` uses these boundaries: `<60s`→`now`, `<60m`→`N min`, `<24h`→`N hr`, `<30d`→`N day`, `<365d`→`N mth`, else `N yr`. Singular abbreviations throughout.
- **Why:** compact enough for a 6–7 cell column, unambiguous, fixed-width keeps the table from jittering.
- **Rejected:** following git's own relative format (too verbose); showing only the largest unit (correct but "now" for a 5-hour-old commit is misleading).

### Settings format: `-F` flag override
- **Decision:** The `-F/--full` flag in log/graph still shows absolute ISO date + full author name, regardless of active format settings. Settings only affect the *default* display.
- **Why:** `-F` is an explicit per-run override whose existing behavior is well-documented. Silently changing it would be surprising.
- **Rejected:** making `-F` respect settings (breaking change); removing `-F` (useful one-shot toggle).

## 2026-08 · Per-view column settings, worktree column, ultra-short branch format (p14)

### Per-view hidden-column lists
- **Decision:** `gint br` and `gint lg` persist their hidden column titles (`branchHiddenColumns`, `logHiddenColumns`) in `settings.json`, toggled from a Display section in the view-aware `:settings` overlay. Hiding is TUI-only — `-I`/`RenderTable` always renders the full column set, and the `Column` struct is untouched.
- **Why:** users want per-view display control without re-running `-s`/`-F`; keeping `-I` stable preserves scripting. Filtering at the cli layer by `Title` via `tui.FilterColumns` reuses the existing density/column machinery, and a new `Config.ColumnsRefresh` hook lets the overlay preview/revert re-filter live.
- **Rejected:** per-column `Hidden` field on `Column` (would leak view state into the pure spec); mutating the shared column slice (aliasing bugs); applying hidden lists to `-I` (breaks scripts and the "two never drift" contract).

### Ultra-short branch format
- **Decision:** a third global `BranchFormat` value `ultra-short`: the segment after the last `/`, vowels stripped (`feat/auth/login-form` → `lgn-frm`). It applies wherever branch names render, exactly like `short`.
- **Why:** very long feature branches still overflow even `short` (`f/a/login-form`); a compact slug keeps tables scannable. Non-vowel characters (digits, `-`, `_`) survive so the result stays recognizable.
- **Rejected:** reusing `short` semantics (still too wide for deeply nested names); dropping separators entirely (unreadable); a per-view format (one global knob matches the existing date/branch/author pattern).

### Worktree-path format
- **Decision:** one global `WorktreePathFormat` (`shortest`/`relative`/`absolute`) shared by the branch, log, and worktree views' path columns. Cells are sourced from `git worktree list` (which includes the main worktree; `%(worktreepath)` is empty there), stored raw, and formatted at render time so settings preview live.
- **Why:** the old `shortestPath` was duplicated per view and couldn't be configured; rendering from the raw path keeps the source-of-truth single and the format a display concern.
- **Rejected:** per-view path formats (one knob is enough); using `%(worktreepath)` for the main worktree (empty); a separate worktree column in `gint wt` (its path column already serves this).

### TUI-only scope for settings
- **Decision:** all p14 settings (hidden columns, worktree path format) affect the interactive views only.
- **Why:** `-I` output is the scripting contract; changing it silently would break consumers. The `-F`/`-s` density flags remain the per-invocation column switch for `-I`.
- **Rejected:** threading hidden columns through `TableOptions`.

### Hiding columns must not filter the List's column set
- **Decision:** hidden-column toggles apply inside `List.visibleColumns()` via a live `Config.HiddenColumns func() map[string]bool` predicate; `l.columns` always stays the full set.
- **Why:** `columnIndex` maps a visible column back to its position in `l.columns`, and `cell()` indexes `Item.Columns()` cells positionally against that set. Pre-filtering the slice (the first p14 implementation, per the original plan) desynced the two — with `branch` hidden, the `last commit` header rendered branch names. The predicate approach also removed the `ColumnsRefresh`/`SetColumns` machinery: toggles preview live for free, exactly like the frozen format vars.
- **Rejected:** filtering at the cli layer before `tui.New` (breaks cell alignment — the bug above); storing a separate Title→index map alongside the filtered set (two sources of truth for one ordering).
