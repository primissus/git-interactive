# Phase 8 — Hardening, polish & v1 release

## Goal

Close the gaps between "commands work" and "v1 done": consistency, docs, test depth, and future-proofing hooks.

## Scope

- **Consistency audit**: every view honors the shared interaction model — shortcuts, confirmations, select mode, `-i/-I/-S/-F/-s` — identically. Fix drift.
- **Future-proofing checks** (design-only, no implementation): select mode exposes a two-item hook for the future `diff` integration; conflict component API covers `resolve-conflicts`; remote-oriented commands (`remote`, `branch-remotes`) wouldn't require re-architecture.
- **Validation**: finalize the input-validation choice (evaluate `go-playground/validator` vs lighter alternatives) and apply it to text inputs (branch names, refs, messages).
- **Test depth**: teatest flows for each command's happy path + one destructive-confirmation path; race detector in CI target.
- **UX polish**: color scheme via Lip Gloss centralized; terminal-resize handling; graceful behavior outside a git repo and in empty repos.
- **Docs & release**: README (install, command tour, keybinding table generated from the shortcut registry if cheap), `make install`, version stamping (`gint --version`).

## Tasks

1. Cross-command consistency audit against PROMPT.md tables; fix findings.
2. Validation library decision + application (record in .context/decisions).
3. Test sweep: per-command teatest happy path + confirmation path; `make test` with `-race`.
4. Edge cases: not-a-repo, empty repo, detached HEAD, terminal resize.
5. README + keybinding reference; version stamping; `make install` verified.

## Acceptance

- A new user can install via README and use every v1 command; test suite covers each command's core flow; `make lint test` clean.

## References

- PROMPT.md → all sections, especially "Shared interaction model" and "Future".
