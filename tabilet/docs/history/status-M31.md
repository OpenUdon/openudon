# Retired milestone M31 - M31 - Trusted Execution Hardening

**Milestone.** M31
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M31.md
**Source status SHA-256.** 47264a44e6b75893740a36fba380fa645ef0a85affd91da259c95e854daa7f1c
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
# M31 - Trusted Execution Hardening

## Goal

Make `openudon run` a stronger, auditable trusted-handoff gate for approved
packages. M31 is execution-handoff hardening only: it does not add public UWS
semantics, provider execution, orchestration service behavior, or udon runtime
implementation.

## Status

| Item | State | Notes |
|---|---|---|
| Milestone framed | `[+]` | M31 follows M30 and focuses on `openudon run`, approval/digest enforcement, executor handoff config, dry-run behavior, and sandbox/production boundary evidence. |
| Strict run-config parsing implemented | `[+]` | `openudon.executor-run.v1` loading now rejects unknown fields, trailing JSON, invalid package digests, missing package paths, direct production flags, unsafe paths, and credential env-name collisions. |
| Dry-run staging implemented | `[+]` | `openudon run --dry-run` now validates runner config, stages the digest-covered package, and verifies the staged digest without requiring credential values or invoking the executor. |
| Run evidence implemented | `[+]` | `openudon.run-evidence.v1` is written to `<workdir>/run-evidence.json` with non-secret gate, package, stage kind, digest, credential binding, and executor invocation evidence, including failed invocation status. |
| Approval parsing hardened | `[+]` | Approval JSON parsing now rejects trailing JSON in addition to unknown fields and existing scope/state/tier/digest/expiry checks. |
| Operator docs updated | `[+]` | README, review handoff, safety, SaaS review handoff, SaaS operator release, architecture, tech-stack, and milestone memory now describe staged dry-runs and run evidence. |
| Deterministic gates checked | `[+]` | Ran focused trusted-runner/udon-runner/package-artifact/CLI tests, `go test ./...`, `go vet ./...`, `go run ./cmd/openudon check-doc-memory`, UWS validation, `make check`, and diff checks for OpenUdon and the tracked memory bank. |

## Design Notes

- Dry-run is still provider-free and credential-value-free, but it now proves
  the same package staging and staged digest invariant used by real handoff.
- Run evidence is OpenUdon-owned operational evidence, not a public UWS schema.
  It must never include credential values, raw environment values, or approval
  notes.
- Evidence uses `stage_kind: preflight` for `OPENUDON_UDON_RUNNER` handoff
  because the external runner owns final executor-visible staging.
- `OPENUDON_EXECUTOR` remains the canonical final executor selector. The
  optional `OPENUDON_UDON_RUNNER` outer shim override remains a trusted operator
  input and is validated as an absolute executable path.
- Production execution remains outside OpenUdon and still requires
  `approved_for_production`, trusted credentials, and a configured executor.
~~~~~~~~~~~~~~~~~~~~
