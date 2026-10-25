# Legacy CLI retirement

A32 retires the private OpenUdon authoring and executor-handoff implementation.
Current OpenUdon supplies [public trust APIs](public-trust-api.md),
[package v3 construction and verification](package-v3.md), and the
[public browser supplement](browser-package-v3.md). The `openudon` and
`udon-runner` executables and their archive publication are removed.

| Retired entrypoints | Current owner or supported path |
|---|---|
| `synthesize`, `build`, `promote`, `assess`, `step`, `authoring`, `flow-review` | Kinet owns product authoring; `packagev3` accepts reviewed UWS/source bytes. Intent/HCL synthesis is retired. |
| `browser-author`, `browser-capture`, `__browsertools-worker` | Kinet owns capture/authoring workers and custody; public receipt verification remains in `browserverify`. |
| `browser-*-eval`, `browser-system-input`, `browser-system-component`, `browser-system-dev` | Retired OpenUdon harnesses; consuming owners qualify their actual runtime integration. |
| `package`, `broker-inspect`, `approval-template`, `simulate` | Public package/trust/approval APIs remain; Kinet owns user flows and preview integration. |
| `run`, `local-udon-smoke`, `udon-runner` | Kinet owns exact confirmation and isolated worker handoff; private Udon owns execution. |
| `run-evidence`, `release-notes` | Public `udonreport`/`runevidence` byte validation remains. CLI signing/archive orchestration retires. |
| `eval`, `readiness`, `release-evidence`, `smoke-matrix`, `n8n-bridge`, `catalog`, repository/validation checks | Legacy orchestration commands retire. Package-local builds, tests and public-boundary checks remain. |

These ownership pointers describe the existing boundaries; they promise no new
Kinet feature or command-for-command compatibility. Existing public success and
refusal tests remain with their public packages. The three public surface/import
regressions formerly under `internal/publicapi` move unchanged to
`tests/publicapi`. Six unique policy/report/authority success and refusal
regressions move from private aliases to `tests/credentialpolicy`,
`tests/udonreport` and `tests/authority`, calling the existing public APIs.
Private artifact-binding exemptions and filesystem sidecar helpers retire.

P10 construction/verification/transaction/approval/evidence, P11 named-output
ownership/deterministic assessment/stale-proof refusal, P12 authored bounded
authentication and readable untimed history, and P13 saved-metadata expiry and
observed fallback-rights checks remain unchanged. Public source files, embedded
schemas, UWS archive and historical wire/source fixtures retain exact bytes.
Private browser candidate lifecycle and intent synthesis tests retire with their
implementation; they supplied no replacement runtime qualification claim.

Earlier guides and examples describe their original pinned revisions. Historical
qualification, publication and inventory records remain immutable. No existing
consumer pin, tag or history is rewritten. The accepted Kinet M53 handoff proves
retirement of the old CLI/image/internal producer path while public libraries
remain live. Remote fetchability and retained consumer builds remain A32.5 gates.

The A32.1 inventory is a frozen pre-removal baseline. Final regeneration and the
no-HCL-input proof are separate assigned steps; this source cut does not claim
those acceptance gates complete.
