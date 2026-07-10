# Phase 1 progress

Status: **done**

- [x] 1. Module init + directory layout
- [x] 2. Cobra root + stub commands with aliases and common flags
- [x] 3. Makefile + golangci-lint; lint/test targets pass
- [x] 4. internal/git runner + parsers (branch, worktree, log, status, stash)
- [x] 5. Fixture-repo test helper + parser tests

## Log

- 2026-07-10: Implemented full phase 1 scope in one pass.
  - `go.mod` (module `git-interact`), layout `cmd/gint`, `internal/cli`, `internal/git`.
  - Cobra root + all 11 v1 subcommands with their aliases and shared flags
    (`-i/-I/-S/-F/-s`); all but `branch -I` are stubs that echo their parsed
    flags/args, per phase scope.
  - `internal/git`: `Runner` (exec wrapper, stderr-carrying `*Error`) plus
    parsers for `branch --format`, `worktree list --porcelain`, `log --pretty`,
    `status --porcelain=v2 --branch` (staged/unstaged/conflicts/untracked/
    ignored, ahead/behind), and `stash list --format`. Field formats informed
    by the `git-br.py`/`git-lg.py`/`git-st.py` references.
  - `branch -I` wired to the real git layer for the phase's smoke-test
    acceptance criterion; the interactive path (`-i`, default) stays a stub
    until phase 2/3.
  - `Makefile` (`build`, `install`, `test`, `lint`, `fmt`) and `.golangci.yml`
    (v2 config, `standard` linters + `unconvert`/`unparam`/`misspell`,
    `gofmt`/`goimports` formatters). `golangci-lint` installed via `brew`.
  - Fixture-repo test helper (`newTestRepo`/`mustGit`/`writeFile` in
    `internal/git/testrepo_test.go`) builds a real repo under `t.TempDir()`
    with deterministic author/committer; parser tests cover branches
    (current-branch marker, subject/author), commits (ordering, refs),
    worktrees (path resolution), stashes (branch/message parsing), and
    status (staged/unstaged/untracked plus a real merge-conflict case).
  - Verified acceptance: `make build && ./gint branch -I` prints a real
    tabular branch list; `make lint test` clean (0 lint issues, all tests
    pass).
