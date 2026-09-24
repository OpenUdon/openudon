# Status E21 — Repair current-stack Udon build and preserve M86 report meaning

**State:** Active; E21.4 is in progress. Synthetic local qualification only.

**Scope boundary:** Do not contact providers or target accounts, run a public
canary, adopt a runtime, or deploy. Publish the reviewed OpenUdon source only
after current-stack qualification and bounded review pass. W8M adoption and
push remain outside E21.

Markers: `[ ]` pending, `[~]` in progress, `[+]` complete, `[!]` blocked,
`[-]` closed history, `[X]` cancelled.

## Tasks

| Item | State | Notes |
| --- | --- | --- |
| E21.1 Freeze M86 current-lock meaning | `[+]` | Copied the exact pre-E21 scenario lock bytes (SHA-256 `57ebe6c70bc0b1e810ed4fb490f36ecb227c47f362bb738b56680ce7abcd6a77`) and integration lock bytes (`9eec17f1489e1c805e2d2bfb8b89a439ee7153d5c49ec09a3ee903ce6761d393`) into versioned v2 snapshots and routed the M86 readers to them. Focused scenario/integration tests pass. M86 integration/loopback/journey reports independently verify at 19/19, 23/23 and 11/11 with their recorded digests unchanged. E21.2 advanced the mutable current locks while retaining these v2 snapshots. |
| E21.2 Pin and emit the repaired current stack | `[+]` | Current scenario/integration locks select Udon `6d32d4967469c579d35adcf47eaddb76a225dbae`; the separate 14-repository closure is exact and checks clean commits before execution. Current scenario/journey and integration contracts emit v3; M86 v2 uses frozen lock snapshots. Focused tests and vet pass; build-input lock SHA-256 is `4993304edf46953c33b6112c4f00e3fcf526811ac91400989b775a19066977b9`. |
| E21.3 Extend native current-stack qualification | `[+]` | Added `browser-system-eval --stack current --suite loopback` and `make browser-system-current-check`; historical remains default v2 and v1/v2 verifiers keep their inventories. Current native v3 routes the 14-source clean closure through the build, scenario, BAP and BRP stages, with clean-source preflight and per-stage rechecks. Focused tests and vet pass. |
| E21.4 Qualify clean current-stack evidence | `[~]` | Attempt 1 used clean OpenUdon `31765b65a3b36e4daf78d5eddb45801b5eb728b8`; the v3 integration report failed 18/19 because Browserdriver's `npm test` could not find the pinned TypeScript compiler in the clean source checkout. Preserve report `/home/peter/.local/state/openudon/e21-current-qualification-20260924/evidence/integration-v3.json` (SHA-256 `b06a0d19a5d4921c0cf34995e260a683fbce4e39cd6c58617b2484d9e1860d30`). Diagnosis confirmed the separately supplied TypeScript 5.9.2 modules are already available. E21.4 adds lock-version validation for that read-only directory and runs the exact npm test source in a disposable clone; focused tests, focused vet, and `make fast` pass. Retry only after committing this correction, with a fresh worktree and output path. |
| E21.5 Review and publish OpenUdon | `[ ]` | Complete all required checks and the bounded review-fix gate with no open P1/P2, record report/source digests and teardown, then push OpenUdon only. Do not push W8M or adopt its candidate. |

## Acceptance

The original M86 integration and two current scenario v2 reports verify with
their recorded digests after the E21 current lock changes. Malformed and
cross-version evidence is rejected. `--stack current` resolves Udon
`6d32d4967469c579d35adcf47eaddb76a225dbae` plus exactly 14 clean locked local
replacements before browser launch. Current scenario, journey, integration
and native reports use v3; v1 historical default and M86 v2 verifiers keep
their original semantics. The 19 integration gates, all 23 loopback cases,
all 11 journey cases, current native scenario/BAP/BRP stages, independent
verification, source cleanliness, and bounded review pass. Only OpenUdon is
published.

## Dependencies and downstream impact

M86 is retired history and supplies the v2 lock/report compatibility contract;
it is verification input, not reopened work. The W8M W20.6 affected candidate
smoke and W21 uncached qualification depend on this published OpenUdon commit
and both current lock digests. No other repository source is changed or
published by E21.

## Review and verification record

Review starts at iteration 1 after automated checks and current qualification.
Record each whole-scope review, findings, fixes and verification here. Preserve
the iteration number across continuation and stop after iteration 10 if any
P1/P2 remains.
