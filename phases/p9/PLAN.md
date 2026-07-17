# Phase 9 — `resolve-conflicts` and per-hunk resolution

## Goal

The `resolve-conflicts` (`rc`) command from PROMPT.md's Future section: a standalone
interactive resolver for merge/rebase/cherry-pick conflicts. Phase 7's component
already covers detection, side labels, whole-file resolution, and
continue/skip/abort; this phase adds what the spec requires beyond it —
per-conflict (hunk) walking, conflict counts, resolved-files progress, orphan
conflicts, and the `status` hand-off.

## Decisions (interview, 2026-07-10)

- **Orphan conflicts**: `gint rc` also works when conflicted files exist with no
  operation in progress (conflicted `git stash pop`, failed `git apply -3`).
  Side labels come from the conflict-marker annotations (e.g. `Updated
  upstream` / `Stashed changes`); continue/skip/abort are hidden since there is
  no operation to drive.
- **Hunk view**: stacked blocks — both sides one above the other, each headed by
  its resolved label; no side-by-side layout.
- **`status` hand-off**: wired in this phase — status's conflict state gains a
  "resolve conflicts" operation that opens this view.
- **Docs**: promoting the spec from PROMPT.md → Future into Commands (updated
  with these decisions) is a task of this phase.

## Scope

### git layer: conflict hunks

- Parse a conflicted file into hunks: for each `<<<<<<<`…`>>>>>>>` block, the
  ours/theirs content, any diff3 base section, the marker annotations (used as
  labels in orphan mode), and the line span in the file.
- Per-hunk resolution: take ours / take theirs / take both rewrites only that
  hunk, leaving the others' markers intact; when a file has no markers left it
  is staged (same behavior the whole-file ops already have).
- Conflict counts per file, for the list column and the walker header.
- Progress baseline: snapshot the conflicted set when the command starts so the
  header can say "2/5 files resolved" (git only reports the still-unresolved
  ones, so resolved-count needs a baseline).

### `resolve-conflicts` (`rc`)

- No operation and no conflicts → plain "nothing to resolve" message.
- In-progress operation → the phase-7 resolver list (reused), extended with a
  conflict-count column and the progress header ("rebasing `feat/x` onto
  `main`, 2/5 files resolved").
- Orphan mode → same list without continue/skip/abort, marker-derived labels.
- **Enter on a file** opens the hunk walker: stacked blocks per the decided
  layout; `o`/`t`/`b` resolve the current hunk (whole-file variants stay
  available from the list), `e` opens `$EDITOR` at the hunk's line,
  `n`/`p` move between conflicts, resolving auto-advances, `esc` returns to the
  file list; resolving a file's last hunk stages it and returns to the list.
- When every file is resolved, offer continue (in-progress mode); abort with
  confirmation available at any point, as today.
- `-I`/`--no-interactive` prints the conflicted files with their conflict
  counts as a table.

### `status` hand-off

- When `status` detects a stopped operation it keeps its inline per-file ops but
  gains a "resolve conflicts" operation that opens this resolver.

## Tasks

1. Hunk model in `internal/git`: parser (marker blocks → ours/theirs/base,
   annotations, line spans; diff3-aware), per-file conflict counts.
2. Per-hunk resolution: rewrite a single hunk (ours/theirs/both), stage the
   file once marker-free; progress baseline snapshot for resolved-files counts.
3. Hunk walker TUI view: stacked labeled blocks, o/t/b/e keys, n/p navigation,
   auto-advance, `$EDITOR` at the hunk's line, esc back to the list.
4. `resolve-conflicts` command: registration + `rc` alias, in-progress /
   orphan / nothing-to-resolve dispatch, count column + progress header on the
   phase-7 list, Enter → walker, offer-continue when clean, `-I` table.
5. Wire the `status` hand-off ("resolve conflicts" operation → this view).
6. Docs: move the `resolve-conflicts` spec from PROMPT.md Future into Commands,
   updated with the decisions above.
7. Tests: parser fixtures (diff3 base, missing trailing newline, marker
   annotations), per-hunk rewrites leaving siblings intact, orphan detection
   (conflicted stash pop), walker interaction via teatest, progress counts.

## Acceptance

- A conflicted rebase, merge, and stash pop are each resolvable hunk by hunk
  entirely inside `gint rc`; sides always carry branch/commit (or marker)
  labels; `status` hands off to the resolver; PROMPT.md lists the command under
  Commands; `make lint test` clean.

## References

- PROMPT.md → `resolve-conflicts` (Future, promoted by task 6), `status`.
- phases/p7 — the component this phase extends (`internal/git/conflict.go`,
  `internal/cli/conflict.go`).
