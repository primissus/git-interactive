# Phase 2 — Shared TUI framework

## Goal

The reusable Bubble Tea interaction layer every command view sits on, implementing PROMPT.md's "Shared interaction model" once so command phases only wire data and operations into it.

## Scope

- **List view component** (Bubbles table/list + Lip Gloss): column layout, viewport-height pagination, arrow + vim (`hjkl`) navigation, current-item highlight with color + dot marker.
- **Fuzzy search** (`/`) over rows via `sahilm/fuzzy`, live-filtering the list.
- **Context menu**: Enter opens an operations menu for the selected item; menu items are supplied by the command. Menu itself supports fuzzy matching on operation names (needed later for `gint branch <name>` disambiguation).
- **Confirmation flows**: plain yes/no confirm; typed confirm (must type an exact phrase like `force`, `delete all`); multi-choice confirm (used later by merge/reset/cherry-pick/commit). One component, parameterized.
- **Select mode** (`Shift+X`): multi-select with visual markers; bulk operations menu provided by the command. Design the selection API so a future two-item `diff` operation can plug in (see PROMPT.md Future).
- **Shortcut registry**: common shortcuts (`/`, `Shift+N/C/D/R`, `p`, `Shift+P`, `Shift+X`) dispatch to the same operations as the menu, including their confirmations.
- **Flag plumbing**: `-i/-I` toggling interactive vs plain tabular print; `-S`, `-F`, `-s` view density/sort hooks the list component understands.
- **Input prompts**: single-line text input component (branch name, commit message, typed confirmations).

## Out of scope

Graph rendering (phase 4), conflict-resolution component (phase 7).

## Tasks

1. Core list model: data-source interface, columns, pagination, navigation, highlight.
2. Fuzzy search overlay.
3. Context menu component with operation registry + fuzzy matching.
4. Confirmation components (yes/no, typed, multi-choice).
5. Select mode + bulk operations plumbing.
6. Shortcut registry mapped onto operations.
7. Non-interactive (`-I`) tabular renderer sharing the column definitions.
8. teatest coverage: navigation, search, menu open/dispatch, typed confirm, select mode.

## Acceptance

- A demo command (can be a hidden `gint _demo`) exercises every behavior above.
- teatest suite green; `make lint test` clean.

## References

- PROMPT.md → "Shared interaction model", "Common shortcuts", "Common flags".
