# Phase 8 progress

Status: **complete**

- [x] 1. Consistency audit
- [x] 2. Validation decision + application
- [x] 3. Test sweep (+ race)
- [x] 4. Edge cases (not-a-repo, empty repo, detached HEAD, resize)
- [x] 5. README, keybindings, version, install

## Log

### 2026-07-10

- **consistency audit** (audited every `internal/cli` command against
  PROMPT.md's shared model):
  - Fixed the one real bug: `log`'s `reset` was bound to `Key: "X"`, which the
    framework reserves for select mode (`tui/list.go`) — a dead shortcut. Reset
    is now menu-only (PROMPT.md assigns it no key).
  - Fixed a cosmetic drift: `graph`/`graph-branch` `-I` output now prints the
    current-commit `*` marker, matching `log`.
  - Reconciled the audit against `.context/decisions.md`: `merge` having no
    `-I` mode is a recorded decision, not drift. Recorded the remaining
    low-severity, intentional divergences (`stash` `p`=pop; `-S` honored only
    by `branch`) and the select-mode diff hook (`List.SelectedItems` + bulk
    ops) as decisions so a future audit won't re-flag them.
- **validation**: rejected go-playground/validator (a struct-tag validator for
  web DTOs — wrong shape for git ref rules) in favor of a tiny in-house
  `internal/validate` package. `BranchName` applies git's ref-format rules
  in-process (no repo needed, precise per-rule messages). Added
  `InputSpec.Validate`/`AllowEmpty` hooks to the shared input component (used by
  both `List` and `Flow`); wired `BranchName` into branch/worktree name inputs
  and `AllowEmpty` into the genuinely-optional inputs (stash message, lock
  reason). Fixed `rebase` reword's misleading "leave blank" placeholder (blank
  would have reworded to an empty subject).
- **tests + race**: `make test` now runs `-race`. Added end-to-end glue tests
  driving `commit` and `merge`'s choice→git mappings against fixture repos
  (`cli/commit_test.go`, `cli/testrepo_test.go`), input validation/`AllowEmpty`
  tests (`tui`), a resize teatest, and the empty-repo `ListCommits` case.
  94 tests pass under `-race`; the shared List's interaction model
  (happy path + typed/bulk destructive confirms) is already covered by the
  framework teatest, so per-command flows aren't duplicated.
- **edge cases**:
  - not-a-repo: root `PersistentPreRunE` fails fast with a clear message
    instead of a raw git error (demo exempt); also fixed a latent double-print
    (`SilenceErrors: true` — main owns error output).
  - empty repo: `ListCommits`/`ListCommitGraph` return an empty history on an
    unborn branch instead of surfacing git's "no commits yet" error.
  - detached HEAD: already handled — `CurrentBranch` returns "" and views show
    "(HEAD detached at …)".
  - resize: already handled by `List`'s `WindowSizeMsg` path; locked in with a
    test.
- **docs & release**: `--version` via a build-time `-ldflags -X main.version`
  stamp (git-describe in the Makefile); `make install` verified; wrote
  `README.md` (install, command tour, keybinding + flag tables).
- `make lint test` clean.
