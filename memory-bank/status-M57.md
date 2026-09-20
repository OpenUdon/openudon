# Status M57 - Trusted Executor Output Contract

State of the trusted-executor output contract, udon report emission, and
OpenUdon async status/read ingestion.

## Goal

Give OpenUdon a real trusted-executor-owned output artifact to ingest into
neutral async evidence sidecars without inventing executor observations.

## Tasks

| Item | State | Notes |
|---|---|---|
| M57.1 udon report contract | `[+]` | Added `--execution-report` and `udon.execution-report.v1` JSON emission in udon execute mode. |
| M57.2 OpenUdon report handoff | `[+]` | OpenUdon passes `--execution-report` to compatible internal udon executor argv and leaves dry-run/external-runner preflight evidence without report paths. |
| M57.3 async status/read ingestion | `[+]` | Real udon reports are translated into Evidence async status observations and confirmation-read observations when output digests exist. |
| M57.4 docs and memory | `[+]` | Review-handoff docs, release docs, architecture, tech-stack, and milestone memory describe the report boundary. |
| M57.5 verification | `[+]` | Focused udon/OpenUdon tests, schema parse, doc-memory, release gates, and diff checks pass; see final verification notes. |

## Acceptance Criteria

- Udon can emit a non-secret execution report for direct workflow execution.
- OpenUdon ingests only real executor reports, not stdout/stderr or inferred
  runtime state.
- Async sidecars can include request, response, status, and confirmation-read
  records while preserving `openudon.run-evidence.v1`.
- External runner shim evidence remains preflight-only unless a real output
  contract is available.

## Verification

```bash
go test ./cmd/udon
go test ./...
go vet ./...
go test ./cmd/openudon ./internal/trustedrunner
go run ./cmd/openudon check-doc-memory
git diff --check
git -C ../tofu diff --check -- openudon
```
