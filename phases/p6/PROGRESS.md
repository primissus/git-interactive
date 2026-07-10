# Phase 6 progress

Status: **done**

- [x] 1. Commit flow component
- [x] 2. `commit` command (amend, no-verify)
- [x] 3. Merge flow + `merge` command
- [x] 4. Wire real flows into status/branch/log stubs
- [x] 5. Tests

## Log

- 2026-07-10: Implemented full phase 6 scope — `commit`, `merge`, and wiring
  the stubs left in phases 3-5.
  - **`internal/tui/flow.go`** (new): `Flow`, a minimal standalone wizard
    (optional message input → multi-choice confirm, with an `EditValue`
    choice that loops back to the input) for commands with no list to
    select from. Reuses `List`'s existing `inputModel`/`confirmModel`
    components. Needed only by the standalone `commit`/`merge` commands —
    see `.context/decisions.md` for why status/branch/log/graph's wiring
    uses plain `List` Input+Confirm operations instead of a nested `Flow`
    (Bubble Tea doesn't support nesting a second `tea.Program` inside a
    running one).
  - **`internal/git`**: new `commit.go` (`CommitStaged`, `AmendCommit`
    with `--no-edit` when no message is given, `IsCommitPushed` — the
    amend "already pushed" check) and `merge.go` (`MergeBranch` with
    `MergeDefault`/`MergeFFOnly`/`MergeNoFF`/`MergeSquash`; squash stages
    then commits directly, since `git merge --squash` alone never commits).
  - **`internal/cli/commit.go`**: `commitConfirmChoices`/`runCommitChoice`
    are shared by the standalone command and status's inline commit/amend.
    `gint commit` runs the `Flow` by default; `-A/--amend` and `-I
    <message>` are direct non-interactive shortcuts bypassing it, matching
    branch/worktree's flag-driven direct-path convention.
  - **`internal/cli/merge.go`**: `mergeConfirmChoices`/`runMergeChoice`
    shared by the standalone command and branch/log/graph's merge-into-
    current. `gint merge <branch>` detects an in-progress merge first and
    offers continue/abort instead of starting a new one (PLAN.md → merge
    task 3). No `-I` mode — decision recorded (a merge's mode choice has
    no safe non-interactive default, unlike a plain commit).
  - **Wiring**: `branch`'s Shift+M, `log`/`graph`'s Shift+M (resolving the
    commit's first local branch ref), and `status`'s `c`/Shift+A now call
    the real `runMergeChoice`/`runCommitChoice` logic instead of returning
    a "not implemented" status. No stub language remains outside `rebase`
    (still phase 7's stub, as scoped).
  - **Tests**: `internal/git` gained fixture-repo tests for every new
    commit/merge function, including a real push-to-a-bare-remote round
    trip for `IsCommitPushed` and default/ff-only/no-ff/squash merge
    outcomes verified by commit count and resulting tree state.
    `internal/tui` gained `Flow` tests covering the input→confirm→Run path
    and the edit-loops-back-to-input path. `internal/cli` gained a
    `firstRef` unit test.
  - Verified acceptance: `./gint commit -I <msg>` and `./gint commit -A
    <msg>` both commit against a real repo (confirmed via `git log`);
    `make lint test` clean (0 lint issues, all tests pass).
