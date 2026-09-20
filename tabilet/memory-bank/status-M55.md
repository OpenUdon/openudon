# Status M55 - Run Evidence Usability Hardening

State of the follow-on trusted-runner evidence usability slice after M54.

## Goal

Make async sidecar evidence easier for operators and archives to consume while
preserving OpenUdon's package/run forwarding boundary.

## Tasks

| Item | State | Notes |
|---|---|---|
| M55.1 operator output | `[+]` | `openudon run` now prints the resolved async evidence sidecar path alongside workflow, config, run evidence, workdir, stage, and digest output. |
| M55.2 archive-friendly refs | `[+]` | `async_evidence_files[].path` is now workdir-relative (`async-evidence.json`) while `RunResult.AsyncEvidencePath` carries the resolved local path for CLI output. |
| M55.3 external failure coverage | `[+]` | Trusted-runner tests now assert external runner failure sidecars record `runner_mode: external-runner`, `stage_kind: preflight`, and `fatal_failure`. |
| M55.4 docs/schema example | `[+]` | Review handoff docs include compact examples for `async_evidence_files` and the `openudon.async-evidence-bundle.v1` request/response shape. |
| M55.5 CLI smoke coverage | `[+]` | `cmd/openudon` now has an end-to-end dry-run smoke over the checked-in OpenRPC fixture that asserts stdout includes the `async:` sidecar path. |
| M55.6 archive relocation coverage | `[+]` | Trusted-runner tests copy `run-evidence.json` and `async-evidence.json` to a new archive directory and verify the relative sidecar ref still resolves and matches the run. |

## Acceptance Criteria

- Operators can see the async sidecar path without opening `run-evidence.json`.
- Archived run workdirs can preserve sidecar refs without absolute-path rewriting.
- External runner success and failure are both covered by async evidence tests.
- Documentation shows the additive run-evidence reference and sidecar bundle shape.
- CLI smoke coverage proves operators see the async sidecar path.
- Archive relocation coverage proves sidecar refs remain usable after moving a run directory.

## Verification

Completed:

```bash
go test ./internal/trustedrunner ./cmd/openudon
go run ./cmd/openudon check-doc-memory
git diff --check
git -C ../tofu diff --check -- openudon
go test ./...
go vet ./...
GOWORK=off go test ./...
GOWORK=off go vet ./...
```
