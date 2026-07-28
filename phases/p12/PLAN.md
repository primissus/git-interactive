# Phase 12 — `branch` rebase operation: interactive rebase from branch list

## Goal

In `gint br` (the interactive branch list), add a **rebase** operation that launches the existing interactive rebase plan view. Selecting a branch and pressing `B` rebases the current branch onto the selected one — the daily "update my branch from `main`" flow — and returns to the refreshed branch list afterwards.

## Decisions

- **Direction: current onto selected** — `gint rebase <current> <selected>` replays current's commits on top of the selected base; you stay on your branch. This matches "rebase into current branch" and is the standard "update my feature branch" workflow.
- **Key: `B`** — free in the branch keymap; remappable via keymap.json (`branch.rebase onto this branch`).
- **No bulk** — rebase is inherently single-target (the interactive plan view is a full-screen single-branch flow).
- **No pre-confirm** — the plan view itself is the confirmation surface (submit asks "Run the rebase?"; quitting the plan view cancels).
- **Quit-and-reenter pattern** — the op exits the branch list program (`tea.Quit`), runs the rebase flow (extracted `runRebaseInteractive`), then re-enters the refreshed branch list. Precedent: `runRebaseSelector`'s loop.
- **Shared helper** — `runRebaseInteractive` is the plan → submit → resolver core, used by both `gint rebase` CLI and the branch view. No behavior change to `gint rebase`.

## Scope

### `CLI` package

- `rebase.go`: extract `runRebaseInteractive(cmd, r, target, base, flags)` — the plan view + commit ops + submit + conflict resolver handoff + outcome message. `runRebase` calls it after arg resolution.
- `branch.go`:
  - `branchViewState.rebaseBase string` — set by the new op, consumed/reset by the handoff loop.
  - New op `"rebase onto this branch"` (key `B`, `ScopeItem`, no `Bulk`) — guards: no row, or selected == current branch (status "already on <branch> — nothing to rebase onto"). Sets `state.rebaseBase` and returns `tea.Quit`.
  - `runBranchRebaseHandoff(cmd, r, state, flags)` — handles the post-quit flow: if in-progress rebase → `runConflictResolver`; otherwise `runRebaseInteractive(cur, base)`. Returns `(reenter bool, err)`. Non-fatal rebase errors propagate; the caller returns them rather than re-entering a potentially broken state.
  - `runBranch` and `runBranchDirectMenu` wrap their `p.Run()` in a for-loop calling `runBranchRebaseHandoff`. Normal quit (rebaseBase empty) exits the loop. Both paths reload items each iteration (tips and current marker change after a rebase).

### Tests

- `branch_test.go`: `TestBranchRebaseOpSetsState` — op on non-current branch sets `state.rebaseBase` and returns `tea.QuitMsg`. `TestBranchRebaseOpGuardCurrentBranch` — op on current branch leaves state empty and returns a status msg (not `tea.QuitMsg`).

### Docs

- `PROMPT.md`: branch section — add Rebate bullet after Merge.
- `phases/README.md`: add p12 row.
- `phases/p12/PLAN.md`, `phases/p12/PROGRESS.md`.

## Tasks

1. `rebase.go`: extract `runRebaseInteractive`.
2. `branch.go`: `rebaseBase` state + rebase op.
3. `branch.go`: `runBranchRebaseHandoff` + loop both entry points.
4. Tests.
5. Docs.
6. `make fmt lint test build` verification.

## Acceptance

- From `gint br` on `main`, select `feat/x`, press `B` → plan view titled `gint rebase main onto feat/x` → submit → rebase runs → back in refreshed `gint br`.
- Selecting the current branch → status error, no quit.
- Rebase in progress + `B` → conflict resolver opens, then back to list.
- `gint rebase` CLI behavior unchanged.
- `make fmt lint test build` clean.
