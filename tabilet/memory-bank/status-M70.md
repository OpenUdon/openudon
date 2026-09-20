# Status M70 - Adaptive Evidence-Grounded iCoT v2

State: Complete

## Goal

Replace fixed one-blocker iCoT authoring with an unlimited dependency-frontier
interview grounded in inspected API-source evidence, while retaining OpenUdon's
three prompt modes and trusted artifact boundary.

## Task Status

| Item | State | Notes |
|---|---:|---|
| Adaptive iCoT v2 delivery | `[+]` | Authoring supplies the versioned graph and frontier-round engine; apitools supplies bounded eight-family local discovery; OpenUdon owns the active/candidate workflow graph, v2 wires, source/remote adapters, unified evidence, proposal approval, draft/final promotion, candidate project section, read-only agent report, migrated support fixtures, docs, and offline verification. `tfconfig` and Ramen are unchanged. |
| Review remediation | `[+]` | Blocks partial interactive discovery, preserves network approval policy across discovery, round-trips safety qualifiers through unified evidence attributes, and rejects conflicting selected source targets before staging. Upstream chronology and sidecar fixes are pinned. |

## Scoped Commits

- apitools M69 local discovery: `9d3c05f`
- Authoring M25 graph/frontier engine: `284c32e`
- OpenUdon M70 adapter/lifecycle/docs: `03f6289`
- apitools security-sidecar remediation: `ef32163`
- Authoring transcript chronology and evidence attributes: `a3ec037`, `18ba806`
- OpenUdon discovery/safety/materialization remediation: `e08955c`
- Tracked memory/evolution update: this tofu commit

## Verification

- apitools: `go test ./...`, `go vet ./...`, `git diff --check`
- Authoring: `go test ./...`, `go vet ./...`, compatibility check, `git diff --check`
- OpenUdon: workspace and standalone `go test ./...` / `go vet ./...`, boundary and doc-memory
  checks, variants validation/coverage, scorecard/report verification, release SaaS checks,
  `mkdocs build --strict`, and `git diff --check`
- Ramen: full tests and compatibility check
- all network tests use local test servers; default gates are provider-free and offline
