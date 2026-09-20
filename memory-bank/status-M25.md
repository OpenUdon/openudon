# Status M25 - First-Class Provider Catalog CLI Integration

Task states use `[ ]` not started, `[/]` in progress, `[+]` complete, and `[!]`
blocked.

| Item | State | Notes |
|---|---|---|
| Upstream apitools catalog docs reviewed | `[+]` | Read `../apitools/AGENTS.md`, `memory-bank/milestone.md`, latest evolution v14, and catalog CLI/API surfaces. |
| OpenUdon catalog command surface selected | `[+]` | Selected operator-facing `list`, `inspect`, `advisory`, `specs`, `security-report`, and `import-openapi`; left refresh/audit/stats maintainer commands upstream. |
| Catalog CLI wrappers implemented | `[+]` | `cmd/openudon` now wraps `apitools/catalog` for read-only provider metadata and direct OpenAPI import. |
| Discovery/Smithy/OpenAPI boundary enforced | `[+]` | `import-openapi` only accepts catalog `openapi` spec references; Discovery, Smithy, Stone, and human-docs entries remain advisory. |
| CLI tests added | `[+]` | Added smoke coverage for catalog help, JSON list, Gmail inspect, example advisory, and Discovery-only import rejection. |
| Operator docs updated | `[+]` | README and authoring/related docs now describe first-class provider catalog commands and advisory boundary. |
| Verification completed | `[+]` | `go test ./...`, `go vet ./...`, `GOWORK=off go test ./...`, `GOWORK=off go vet ./...`, `make check`, `check-doc-memory`, catalog smoke commands, and diff checks passed. Stripe direct OpenAPI import smoke passed under `/tmp`. |
