# Status M56 - Run Evidence Verification And Release Readiness

State of the run-evidence verifier, sidecar schema, release-readiness pass, and
executor status/read ingestion review after M55.

## Goal

Make M54/M55 async sidecar output operator-verifiable while preserving the
trusted executor boundary and avoiding invented status or confirmation-read
observations.

## Tasks

| Item | State | Notes |
|---|---|---|
| M56.1 release-readiness baseline | `[+]` | `make check` and `make release-check` pass after M54/M55. |
| M56.2 sidecar verifier command | `[+]` | Added `openudon run-evidence verify --file run-evidence.json` to verify run evidence, relative sidecar paths, sidecar SHA-256 digests, record counts, and async record shapes. |
| M56.3 strict sidecar validation | `[+]` | Verifier rejects unsafe sidecar paths, digest mismatches, record-count mismatches, unknown sidecar fields, unknown record kinds, missing request/response payloads, and mismatched record payloads. |
| M56.4 sidecar schema artifact | `[+]` | Added `docs/schemas/openudon.async-evidence-bundle.v1.schema.json` and linked it from review-handoff docs. |
| M56.5 CLI smoke coverage | `[+]` | Existing dry-run CLI smoke now also runs `openudon run-evidence verify` against the generated evidence. |
| M56.6 executor status/read ingestion review | `[+]` | Closed by M57-M64. OpenUdon now ingests real udon-owned `udon.execution-report.v1` status/read observations into async sidecars and still refuses fake observations or Ramen convergence semantics. |

## Acceptance Criteria

- Operators can verify `run-evidence.json` and async sidecar integrity with a CLI command.
- Sidecar bundle schema is documented as a machine-readable artifact.
- Malformed sidecars fail closed in tests.
- Release-readiness gates pass.
- OpenUdon still forwards package/run evidence only and does not interpret Ramen convergence.

## Verification

```bash
make check
make release-check
go test ./cmd/openudon ./internal/trustedrunner
go run ./cmd/openudon run-evidence verify --file /tmp/openudon-run-smoke/run-evidence.json
jq empty docs/schemas/openudon.async-evidence-bundle.v1.schema.json
mkdocs build --strict --site-dir /tmp/openudon-mkdocs-m56
go run ./cmd/openudon check-doc-memory
git diff --check
git -C ../tofu diff --check -- openudon browsertools
go test ./...
go vet ./...
GOWORK=off go test ./...
GOWORK=off go vet ./...
```
