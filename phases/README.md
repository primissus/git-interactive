# Development phases

Phased plan to build `gint` per [PROMPT.md](../PROMPT.md). Each phase has a `PLAN.md` (goal, scope, tasks, acceptance) and a `PROGRESS.md` (checklist + dated log). Phases are sequential; each ends with `make lint test` clean.

| Phase | Delivers |
|---|---|
| [p1](p1/PLAN.md) | Scaffolding: Cobra skeleton, Makefile, lint, git shell-out layer |
| [p2](p2/PLAN.md) | Shared TUI framework: list, search, menus, confirmations, select mode |
| [p3](p3/PLAN.md) | `branch`, `worktree` |
| [p4](p4/PLAN.md) | `log`, `graph`, `graph-branch` |
| [p5](p5/PLAN.md) | `status`, `add`, `stash` |
| [p6](p6/PLAN.md) | `commit`, `merge` + wiring into earlier views |
| [p7](p7/PLAN.md) | `rebase` + reusable conflict-resolution component |
| [p8](p8/PLAN.md) | Hardening, polish, docs, v1 release |
| [p9](p9/PLAN.md) | `resolve-conflicts`: per-hunk resolution, orphan conflicts, `status` hand-off |
