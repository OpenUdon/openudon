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
