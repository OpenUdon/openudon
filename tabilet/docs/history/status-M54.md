# Retired milestone M54 - Status M54 - Async Evidence Sidecar Forwarding

**Milestone.** M54
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M54.md
**Source status SHA-256.** 63e0255da038e026b4bfff36ec0d0058232b38c1c96316a9fcf3801b07f852bf
**Source milestone snapshot.** tabilet/docs/history/milestone-before-legacy-retirement.md.txt
**Snapshot SHA-256.** 26884eeda9af4ded30e6d545a33dca84c401d2320c22355fe4fb0336d5dcac71
**Evidence.** 71a4f78afbcf2180fc478ffa89c53544c9160648
**Worktree.** includes uncommitted changes
**Review.** not established
**Review iterations.** not recorded
**Verification.** Original status bytes and full earlier milestone bytes preserved by SHA-256; no fresh acceptance claim.
**Consolidated into.** Current milestone dashboard and maintained memory-bank guidance; full earlier text remains in the frozen snapshot.

## Status record

~~~~~~~~~~~~~~~~~~~~markdown
# Status M54 - Async Evidence Sidecar Forwarding

State of OpenUdon's first consumer of `github.com/OpenUdon/evidence/async`.

## Goal

Write neutral async execution evidence sidecars during `openudon run` and
reference them from `openudon.run-evidence.v1` without changing OpenUdon's
trusted-runner ownership boundary.

## Tasks

| Item | State | Notes |
|---|---|---|
| M54.1 dependency update | `[+]` | `github.com/OpenUdon/evidence` now points at the pushed async-record pseudo-version. No committed local `replace` was added. |
| M54.2 run-evidence extension | `[+]` | `trustedrunner.RunEvidence` now has optional `async_evidence_files` refs with path, digest, record count, and purpose. `RunEvidenceVersion` stays `openudon.run-evidence.v1`. |
| M54.3 sidecar bundle | `[+]` | `openudon run` writes one `<workdir>/async-evidence.json` bundle with version `openudon.async-evidence-bundle.v1`. |
| M54.4 neutral async records | `[+]` | Each run sidecar contains an `async.ExecutionRequest` and `async.ExecutionResponse` for the OpenUdon package handoff. Successful dry-run preparation or executor invocation records `accepted`; invocation failure records `fatal_failure`. |
| M54.5 safety boundary | `[+]` | Sidecars carry package/run forwarding metadata only: package scope, staged UWS workflow path, runner mode, stage kind, tier, dry-run flag, run-config path, non-secret argv metadata, reviewer, and package digest. They do not include credential values, raw stdout/stderr, Ramen resource addresses, desired hashes, convergence outcomes, or state semantics. |
| M54.6 tests | `[+]` | Trusted-runner tests cover dry-run sidecar references, accepted internal-runner responses, fatal-failure responses, external runner argv metadata, credential-value non-leakage, and sidecar digest matching. |

## Acceptance Criteria

- `openudon run --dry-run` writes `run-evidence.json` and
  `async-evidence.json`.
- Non-dry successful runner handoff writes request plus accepted response
  records.
- Runner invocation failure still writes run evidence and records a
  fatal-failure async response.
- `run-evidence.json` references sidecar bytes by SHA-256 digest.
- OpenUdon forwards neutral execution evidence only and does not interpret
  Ramen convergence or mutate Ramen state.

## Verification

Completed:

```bash
go test ./internal/trustedrunner
go run ./cmd/openudon check-doc-memory
git diff --check
git -C ../tofu diff --check -- openudon
go test ./...
go vet ./...
GOWORK=off go test ./internal/trustedrunner
GOWORK=off go test ./...
GOWORK=off go vet ./...
```
~~~~~~~~~~~~~~~~~~~~
