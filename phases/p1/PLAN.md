# Phase 1 — Project scaffolding & git layer

## Goal

A buildable, testable, lintable Go project with the Cobra CLI skeleton for every v1 command (stubs), and a git execution layer the rest of the tool builds on.

## Scope

- Go module `git-interact`, binary named `gint`.
- Cobra root command plus stub subcommands with their aliases: `branch`/`br`, `worktree`/`wt`, `graph-branch`/`grb`, `log`/`lg`, `graph`/`gr`, `rebase`/`reb`, `merge`/`mrg`, `status`/`st`, `add`, `stash`/`sth`, `commit`/`cm`.
- Common flags registered as persistent/shared flags: `-i/--interactive` (default), `-I/--no-interactive`, `-S/--sort`, `-F/--full`, `-s/--short`. Stubs just echo what they would do.
- `internal/git` package: shell out to the `git` binary. Typed helpers for the porcelain output we'll need (branch list with commit/author/date, worktree list, log, status --porcelain=v2, stash list). Errors carry stderr.
- `Makefile` with `build`, `install`, `test`, `lint`, `fmt` targets; `golangci-lint` config.
- Unit tests for the git layer (against a fixture repo created in `t.TempDir()`).

## Out of scope

Any TUI code (phase 2), real command behavior (phases 3+).

## Tasks

1. `go mod init`, directory layout (`cmd/gint`, `internal/cli`, `internal/git`).
2. Cobra root + all stub commands with aliases and common flags.
3. Makefile + golangci-lint setup; `make lint` and `make test` pass.
4. `internal/git`: runner (exec wrapper with context, stderr capture) + parsers for branch/worktree/log/status/stash listings.
5. Fixture-repo test helper and tests for each parser.

## Acceptance

- `make build && ./gint branch -I` runs (stub output is fine for -i for now, but `-I` should print a real tabular branch list from the git layer as a smoke test).
- `make lint test` clean.

## References

- PROMPT.md → “Architecture”.
- Python references for output formats: `~/src/env/src/common/git/git-br.py`, `git-lg.py`, `git-st.py`.
