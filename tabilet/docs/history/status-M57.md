# Retired milestone M57 - Status M57 - Trusted Executor Output Contract

**Milestone.** M57
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M57.md
**Source status SHA-256.** fd973b52e63fd09ed5c38c092a39db335f0539ff37c0b055742bbc1d4b7eca16
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
~~~~~~~~~~~~~~~~~~~~
