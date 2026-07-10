# Code conventions

## Style
- Formatting: gofmt (`make fmt`)
- Lint: golangci-lint (`make lint`)
- Naming: standard Go; `[PENDING: file/package naming specifics once code exists]`

## Patterns we DO use
- Shell out to `git`; errors from git carry stderr content.
- One shared TUI layer (phase 2) — commands supply data sources and operation registries, never reimplement navigation/search/confirmation.
- Confirmations: destructive ops always confirm; dangerous ops require typed phrases (`force`, `delete all`, `reset hard`, `clear all`).
- Every list view supports both interactive (`-i`) and plain tabular (`-I`) rendering from the same column definitions.

## FORBIDDEN patterns
- No destructive git operation without a confirmation flow.
- No raw "ours"/"theirs" labels in conflict UIs — always resolve to branch/commit names (rebase inverts them).
- No per-command reimplementation of shared interactions (shortcuts, select mode, search).

## Tests
- Where they live: alongside packages, standard `go test`; TUI flows via teatest.
- What must always be tested: porcelain parsers (fixture repos in `t.TempDir()`), each destructive confirmation path.

## Commits
- Format: `[PENDING: pick a convention — repo history so far has plain messages]`
