# Phase 11 — `branch` ergonomics: copy, group tree, runtime sort, `:` palette, rename pre-fill

## Goal

- `c` copies the selected branch name.
- `T` toggles a collapsible `/`-based tree grouping (enter expands/collapses a dir; leaf-only names with indentation; `G` stays jump-to-bottom everywhere).
- `S` cycles the runtime sort mode (last-commit → created → author → off), updating the title.
- `:` opens a command palette over the operation menu with autocomplete + built-in `:quit` (shared framework feature).
- Rename (`R`) pre-fills the input with the current branch name for easier editing.

## Decisions (interview, 2026-07-23)

- **`G` kept as jump-to-bottom everywhere** — tree-grouping key is `T` (tree). Also reachable via `:group`.
- **Sort cycles** — `S` steps through modes with a status line, no chooser. Title label updates.
- **Collapsible recursive tree** — `/` splits build a prefix tree; groups/dirs sort alphabetically first, then branches in active sort order. Branches show only the leaf segment, indented 2 spaces/depth. Groups implement `DefaultActioner` → enter toggles collapse.
- **`:` palette is shared framework** — opens the operation menu with a `:` prompt, autocomplete suggestions, and a trailing synthetic `quit` entry. Designed to align with p10 task 2 (same `Chrome.CommandPrompt`, same menu-open mechanic). Whichever phase merges first owns the plumbing; the phase that merges second rebases with minimal conflict.
- **Rename pre-fill → `InitialFrom`** — `InputSpec` gains `InitialFrom func([]Item) string`, resolved at invocation time so the op can read the current branch name and pre-fill the text field.

## Scope

### Framework

- `InputSpec.InitialFrom` (operation.go) + resolution in `list.go` `runInput`.
- `SetSortLabel(string) tea.Cmd` + `sortLabelMsg` handling in `list.go` `Update`.
- `:` command palette: `menuModel` command variant (prompt `": "`, `ShowSuggestions`), `List` binding in `updateList`, `Chrome.CommandPrompt` + `Chrome.CommandPlaceholder`, synthetic quit builtin.

### `branch` command

- Copy-name op (`c`) — `copyToClipboard(b.b.Name)`.
- Rename pre-fill — `InitialFrom` returns the branch name.
- Sort-cycle op (`S`) + `branchViewState` (mutable state shared with refresh closure).
- Tree grouping (`T`) + `branch_tree.go` (pure `treeify`, `branchGroupItem`, collapse map).
- Grouping interacts correctly: `-I` table mode and direct-menu (`gint br <name>`) are untouched.

### Tests

- `branch_tree_test.go`: treeify shape, ordering, collapse, leaf display, branch-that-is-also-dir, create-row pinning.
- `branch_test.go`: copy-name clipboard stub, sort cycle order + label, rename InitialFrom, grouping toggle.
- `tui`: `SetSortLabel` msg; `InitialFrom` resolution; palette prompt, quit, suggestion accept.

### Docs

- PROMPT.md: branch section + shared model `:` palette.
- `.context/decisions.md`: above decisions.
- `.context/glossary.md`: "branch grouping (tree mode)", "command palette (`:`)".
- `phases/README.md` table row.

## Tasks

1. Framework: `InitialFrom`, `SetSortLabel` + label msg, `SetViewInfo` msg.
2. Branch: copy-name op (`c`), rename `InitialFrom`.
3. Branch: runtime sort-cycle op (`S`) + `branchViewState` refactor.
4. Branch: tree grouping (`branch_tree.go` + `T` op + collapse).
5. Framework: `:` command palette (command mode on menuModel, List binding, chrome, suggestions, quit).
6. Tests.
7. Docs.

## Acceptance

- `c` copies the branch name; `T` toggles tree view (enter collapses/expands dirs); `S` cycles sort; `:` opens palette with autocomplete (`:s`/`:g`/`:q`); rename opens pre-filled.
- `G` still jumps to bottom everywhere.
- `make fmt lint test build` clean.
