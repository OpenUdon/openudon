# M64 Status - Release Evidence Workflow Consolidation

State markers and commit rules are defined in [milestone.md](milestone.md).

| Item | State | Notes |
|---|---|---|
| Single release-evidence entrypoint | `[+]` | Added `openudon release-evidence` and `make release-evidence` to run local udon smoke, archive verification, release-note draft generation, and gate capture. |
| Summary artifact | `[+]` | Emits `openudon.release-evidence-summary.v1` JSON plus Markdown with commit, paths, gates, verifier output, archive refs, and sidecar/report counts. |
| Documentation | `[+]` | README and release stewardship docs describe the local-only flow and confirm artifacts remain ignored unless explicitly promoted. |
| Verification | `[+]` | Passed focused CLI tests, `go test ./...`, `go vet ./...`, `make check`, `check-doc-memory`, boundary check, and diff checks. |
