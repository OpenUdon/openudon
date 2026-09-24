# Retired milestone M58 - Status M58 - Run Evidence Verifier Hardening

**Milestone.** M58
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M58.md
**Source status SHA-256.** 10a2ec61d9298053dd9c8da24f925b0d9a4c83c6737c3065ea6cce1d507b737f
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
# Status M58 - Run Evidence Verifier Hardening

State of archival and malformed-sidecar verifier hardening after M57.

## Goal

Make `openudon run-evidence verify` fail closed on common archive and sidecar
integrity mistakes while accepting the expanded async observation records.

## Tasks

| Item | State | Notes |
|---|---|---|
| M58.1 expanded sidecar schema | `[+]` | Schema and Go validation now accept execution request/response, status observation, and confirmation-read observation records. |
| M58.2 archive verifier cases | `[+]` | Tests cover archive verification, missing sidecars, duplicate refs, digest/count mismatches, unsafe paths, and bad purposes. |
| M58.3 malformed observation cases | `[+]` | Verifier rejects unknown fields, mismatched payloads, invalid versions, and unknown record kinds. |
| M58.4 docs and memory | `[+]` | Review-handoff docs and memory describe expanded sidecars and archive verification. |
| M58.5 verification | `[+]` | Focused verifier tests, schema parse, doc-memory, release gates, and diff checks pass; see final verification notes. |

## Acceptance Criteria

- The verifier validates all async record kinds OpenUdon can emit.
- Archived run-evidence bundles verify from their archive directory.
- Bad sidecars, duplicate references, and malformed observation records fail
  with actionable errors.

## Verification

```bash
go test ./cmd/openudon ./internal/trustedrunner
go run ./cmd/openudon check-doc-memory
git diff --check
git -C ../tofu diff --check -- openudon
```
~~~~~~~~~~~~~~~~~~~~
