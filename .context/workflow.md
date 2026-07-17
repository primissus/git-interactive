# Workflow

## Before touching anything
1. Read the current phase's `phases/pN/PLAN.md` and `PROGRESS.md`; work phase by phase (p1 → p9).
2. Check PROMPT.md for the exact behavior spec of the command you're touching.
3. Check `.context/decisions.md` before re-deciding stack or design questions.

## To make a change
1. Implement against the phase plan's task list.
2. Add/extend tests (parsers: fixture repos; TUI: teatest).
3. `make fmt lint test`.
4. Tick the task in `phases/pN/PROGRESS.md` and append a dated log note.

## Before calling something done
- [ ] `make build` passes
- [ ] `make lint test` clean
- [ ] Destructive operations have confirmation coverage
- [ ] Behavior matches PROMPT.md (and the Python reference script, where one exists)
- [ ] PROGRESS.md updated

## Deploy
No deploy — a local CLI. `make install` installs the `gint` binary. [PENDING: release/versioning process, phase 8]
