# P06 Status - Advisory Content-Trust Review Evidence

State markers and commit rules are defined in [milestone.md](milestone.md).

## State

Completed at OpenUdon `3c9d0d9`; E12 subsequently completed at `cc378be`.
The user later authorized ordered publication, and OpenUdon is published
through hosted-CI follow-up `2c99fde`. No tag, release, or trusted-runner
authorization was performed.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| P06.1 Add explicit package analyzer entry point | `[+]` | OpenUdon `200c9aa` adds a package-local explicit analyzer entry that returns an empty report for declaration-free legacy packages, stable-reads contained browser profiles, routes valid sources through Browsertools M28, converts unavailable/invalid profile contracts into core fixed-message resolver-failure findings, and propagates cancellation. Determinism, untrusted browser control, resolver failure, no-leak messages, legacy opt-out, focused tests, vet, and diff checks pass. It is not called from `ValidateForExecution` or runtime construction. Task review iteration 1 found no P1/P2-or-higher issue. |
| P06.2 Surface warnings and value-free review evidence | `[+]` | OpenUdon `e23e559` calls the explicit analyzer after ordinary UWS assessment, maps every stable finding to a `warn` check while preserving analyzer severity/path/fixed message, and writes the same ordered value-free fields to a dedicated review section. A clean declared report adds one pass check; declaration-free packages add nothing. Focused tests prove warnings leave overall quality `pass`, review evidence is advisory, legacy checks are unchanged, cancellation propagates, and no underlying load/resolver text is copied. Trusted-runner, approval, and executor code are untouched. Task review iteration 1 found no P1/P2-or-higher issue. |
| P06.3 Document and close the advisory package contract | `[+]` | OpenUdon `3c9d0d9` documents warning-only assessment, Browsertools resolver use, provenance-versus-capability, value-free review evidence, declaration-free compatibility, and the unchanged approval/execution boundary. Focused synthesis and race tests, complete workspace and standalone tests/vet, `make check`, strict MkDocs, and source/memory diff checks pass. Milestone review iteration 1 found no P1/P2-or-higher issue across `200c9aa`, `e23e559`, and `3c9d0d9`. |

## Compatibility And Non-Goals

- Analyzer findings are visible review evidence and never ordinary validation
  errors in this milestone.
- P06 does not add Udon as a production dependency or change Udon report
  versions, trusted-runner policy, plan construction, or executor behavior.
- No runtime values, content excerpts, prompts, credentials, or private paths
  may appear in analyzer-derived evidence.

## Verification

- Focused quality, review, package, and trusted-runner tests.
- `go test ./...`, `go vet ./...`, and proportionate race tests.
- `GOWORK=off go test ./...` and `GOWORK=off go vet ./...`.
- `make check`, strict documentation build, and `git diff --check` in both
  repositories.
- Bounded deep-review gate with no unresolved P1/P2-or-higher finding.
