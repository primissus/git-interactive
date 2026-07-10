# Phase 2 progress

Status: **done**

- [x] 1. Core list model (columns, pagination, navigation, highlight)
- [x] 2. Fuzzy search overlay (`/`)
- [x] 3. Context menu + operation registry with fuzzy matching
- [x] 4. Confirmation components (yes/no, typed, multi-choice)
- [x] 5. Select mode + bulk operations
- [x] 6. Shortcut registry
- [x] 7. Non-interactive tabular renderer (`-I`)
- [x] 8. teatest coverage

## Log

- 2026-07-10: Implemented full phase 2 scope — the shared `internal/tui`
  Bubble Tea framework — in one pass.
  - **Deps**: bubbletea v1.3.10, bubbles v0.21.0, lipgloss v1.1.0,
    sahilm/fuzzy v0.1.1, and x/exp/teatest for TUI tests.
  - **`internal/tui` package**:
    - `item.go` — `Item` interface (`Columns`/`FilterValue`/`Current`),
      `Column` (min/max width, flex, density gating), `Density`
      (short/normal/full). `DensityNormal` is the zero value so an
      unconfigured view renders normally; columns default to "shown at
      normal+full, hidden at short".
    - `list.go` + `render.go` — the `List` model: state machine over
      list/search/menu/confirm/input modes with an orthogonal select-mode
      flag. Arrow + `hjkl` navigation, `less`-style pagination to viewport
      height, current-item highlight + `●` dot marker, flex column layout
      shared with the tabular renderer.
    - `search.go` — fuzzy `/` filter over `FilterValue` via sahilm/fuzzy.
    - `menu.go` — context menu with its own fuzzy filter over operation
      names (the mechanism behind `gint branch <name>` disambiguation);
      `Config.OpenMenu` pre-fills it on start.
    - `confirm.go` — one component for yes/no, typed-phrase, and multi-choice
      confirmations, with per-choice typed escalation (choosing "hard"
      requires typing `reset hard`). Arrows navigate; letter keys select
      directly so a choice can bind `h` without colliding with vim motions.
    - `input.go` — single-line text prompt (bubbles/textinput) for branch
      names, commit messages, typed confirmations.
    - `operation.go` — `Operation`/`Scope`/`OpContext`, `Bulk`/`BulkOnly`
      availability, `Status`/`SetItems` framework commands. The same
      `Operation` backs both the menu entry and its shortcut.
    - `table.go` — `RenderTable` for `-I`, sharing the exact `Column`
      definitions so interactive and plain output never drift.
    - `styles.go` — adaptive Lip Gloss theme (light/dark).
  - **Select mode**: `Shift+X`, space to toggle, `●`/checkbox markers,
    `SelectedItems()` returns the selection in list order — the diff-ready API
    a future two-item `diff` operation reads (PROMPT.md Future).
  - **Demo/acceptance**: hidden `gint _demo` command wires the framework with
    sample branch data and operations covering every confirmation shape and
    flow; honours `-i/-I` and `-F/-s` density. `tui.Demo()` is shared by the
    command and the teatest suite.
  - **Tests**: teatest drives navigation, search (+esc restore), menu
    open/dispatch, menu fuzzy match, shortcut→typed-confirm, input flow, and
    select-mode bulk typed-confirm; plus unit tests for the confirm/input
    components and the tabular renderer's density/marker/truncation. 23 tests
    green; `make lint test` clean.
