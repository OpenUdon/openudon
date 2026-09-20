# Status M66 - Provider Expansion Review

State of the provider-adapter expansion review after the Ramen conversion
migration.

## Goal

Make the provider-expansion backlog explicit without moving desired-state
conversion ownership back into OpenUdon.

## Tasks

| Item | State | Notes |
|---|---|---|
| Diagnostic review | `[+]` | Reviewed the active milestone state: OpenUdon no longer owns desired-state conversion, parser imports, or provider/resource operation mapping. |
| Expansion decision | `[+]` | Provider conversion work belongs in Ramen; OpenUdon may only review/package UWS-facing artifacts generated elsewhere. |
| Boundary update | `[+]` | Milestone notes now state that OpenUdon provider expansion is parked unless a future task is about OpenUdon package/review evidence, not conversion semantics. |

## Acceptance Criteria

- OpenUdon does not regain desired-state conversion behavior.
- Provider expansion decisions route to Ramen unless they only affect
  OpenUdon-owned review/package/handoff evidence.
- The boundary is explicit enough that future work does not restart provider
  conversion in OpenUdon by accident.

## Verification

- `go run ./cmd/openudon check-doc-memory`
- `go run ./cmd/openudon check-apitools-boundary`
- `git diff --check`
- `git -C ../tofu diff --check -- openudon`
