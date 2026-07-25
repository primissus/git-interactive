# Phase 10 — `rebase` target/base, branch selector, in-progress status

## Goal

Make `gint rebase` the fully interactive `git rebase -i`: explicit target/base
branches (positional args or an interactive branch selector), a vim-style `:`
command palette in the shared List, a `--commits` dry-run preview rendered like
`gint log`, and a richer in-progress status view.

## Decisions (interview, 2026-07-23)

- **Arg order is target-first**: `gint rebase <target> [<base>]` — the inverse
  of `git rebase <base> <branch>`. Base defaults to the **current branch**
  (HEAD sha when detached) with one arg.
- **Upstream auto-detection is removed**: every path now has an explicit base.
  `PlanRebase`, `rebaseWindow`, `rootCommit`, and `RebasePlan.Root` go away.
- **`:` command palette replaces Shift+Enter**: Shift+Enter is undetectable in
  bubbletea v1.3.10 (no kitty keyboard protocol). `:` opens the existing fuzzy
  operation menu with a `:` prompt, framework-wide; `:a`+Enter runs `apply`.
- **Selector confirm via external Flow**: after `apply`, a `tui.Flow` asks
  `Rebase <target> onto <base>?` y/N (default N); N/esc relaunches the
  selector with marks preserved (marks live in command-side state). `q` in the
  selector aborts the command.
- **continue/skip/abort all confirm** with direct keys `c`/`s`/`a` (continue
  previously had no confirm). `continueSkipAbortOps` is only used by the
  standalone resolver, so `status` is unaffected.
- **`--commits` is preview-only**: shows the `base..target` commits rendered
  like `gint log`, then exits. No rebase starts; read-only list (no
  operations).

## Scope

### git layer

- `PlanRebaseRange(ctx, r, target, base)`: commits = `git log base..target`
  (existing `ListCommitsRange`). `RebasePlan` gains `Target`, drops `Root`.
- `RunRebasePlan`: appends `target` to `git rebase -i <base> [target]` when
  set; empty `Target` keeps today's HEAD behavior (existing tests unchanged).
- `RebaseProgress` gains `StoppedSHA`/`StoppedSubject`, read from
  `REBASE_HEAD` (`git log -1`), tolerating absence.
- New `ChangedFiles(ctx, r, sha)`: `git diff-tree --no-commit-id --name-only -r`.

### `:` command palette (framework)

- `menuModel` prompt variant (`:` instead of `/`); `List` binds `:` in
  modeList to open the context menu (item ops, or bulk ops in select mode)
  with the command prompt.
- `Chrome.CommandPrompt` (`": "`, keymap.json-overridable) + a `:` row in the
  `?` help overlay.

### `rebase` command

- `Use: "rebase [target] [base]"`, max 2 args, both must be existing local
  branches, `base != target` (clear errors).
- In-progress guard stays first — a stopped rebase always resumes.
- **No args → branch selector**: shared `tui.List` of branches; ops
  `select base` (`B`) / `select target` (`T`) overwrite freely but reject
  `base == target`; marked rows render `(base) <name>` / `(target) <name>`;
  `apply` (ScopeList, via Enter-menu or `:`) validates both marks → confirm
  Flow loop → plan view. Plan view title: `gint rebase <target> onto <base>`.
- **`--commits`**: preview the range like `gint log` (extract `commitItems`
  from `loadLogItems`; new `loadCommitRangeItems`), then exit. `-I` prints the
  table (scriptable). No args + `--commits`: selector opens; `apply` skips the
  y/N confirm and opens the preview. Empty range → `nothing to rebase`.
- `-I` with no args (and no `--commits` selector possible) → error
  `rebase -I requires a target branch`.
- When `target` isn't checked out, `git rebase -i <base> <target>` checks it
  out; final line notes `now on <target>`.

### In-progress status view

- `continue(c)` / `skip(s)` / `abort(a)` — all with y/N confirms.
- Title: `gint rebase — replaying <branch> onto <onto> (N/M) — stopped at
  <sha> "subject"`.
- Conflicts → conflicted-files list as today. Edit-stop (no conflicts) → rows
  are the stopped commit's changed files; per-file resolution ops hidden.

### Tests

- git: `PlanRebaseRange` commit set on a diverged fixture; stopped fields in
  `ReadRebaseProgress`; `ChangedFiles`. Existing `RunRebasePlan` tests keep
  compiling (`Target` defaults to `""`).
- cli: selector row rendering (`(base)`/`(target)` prefixes), mark ops
  (overwrite, `base == target` rejection); `loadCommitRangeItems`.
- tui (teatest): `:` opens the menu with the `:` prompt.

### Docs

- PROMPT.md: shared model gains the `:` palette; `rebase` section rewritten.
- `.context/decisions.md`: the decisions listed above.
- `.context/glossary.md`: "command palette (`:`)".

## Tasks

1. git layer: `PlanRebaseRange`, `RebasePlan.Target`, `RunRebasePlan` target,
   stopped-commit progress, `ChangedFiles`.
2. Framework `:` palette (menu prompt variant, List binding, chrome).
3. `rebase` args + validation + `--commits` preview + plan title.
4. Branch selector + confirm-Flow loop.
5. In-progress status view (keys/confirms, stopped-at title, edit-stop files).
6. `log.go` `commitItems` extraction + `loadCommitRangeItems`.
7. Tests.
8. Docs (PROMPT.md, decisions, glossary, PROGRESS.md).

## Acceptance

- `gint rebase`, `gint rebase <t>`, `gint rebase <t> <b>`, and
  `gint rebase … --commits [-I]` all behave per this plan; selector round-trip
  with declined confirm preserves marks; in-progress view offers c/s/a with
  confirms and shows the stopped commit; `make fmt lint test build` clean.

## References

- PROMPT.md → `rebase`; `.context/decisions.md` (phase-7 rebase entries this
  phase supersedes: base selection, arg-less planning).
