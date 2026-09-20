# Status M59 - Async Evidence Release Candidate Pass

State of release-candidate evidence for the expanded async/run-evidence story.

## Goal

Produce a deterministic local release-readiness snapshot for OpenUdon's
executor report, async sidecar, and run-evidence verification path.

## Tasks

| Item | State | Notes |
|---|---|---|
| M59.1 release docs | `[+]` | Release stewardship and release-note template now cover executor reports, async sidecars, and verifier evidence. |
| M59.2 deterministic release gates | `[+]` | Provider-free OpenUdon checks pass; see final verification notes. |
| M59.3 sidecar archival smoke | `[+]` | Existing archive smoke plus report-backed verifier coverage demonstrate archive verification. |
| M59.4 memory and review | `[+]` | Memory updated for M57-M59; no evolution bump needed because this advances the existing executor-evidence direction. |

## Acceptance Criteria

- Release docs explain how to archive and verify run evidence, async sidecars,
  and executor reports.
- Provider-free gates pass.
- No tag or push is performed unless explicitly requested by the operator.

## Verification

```bash
make check
make release-check
make release-saas-check
mkdocs build --strict --site-dir /tmp/openudon-mkdocs-m59
go run ./cmd/openudon check-doc-memory
git diff --check
git -C ../tofu diff --check -- openudon
GOWORK=off go test ./...
GOWORK=off go vet ./...
```
