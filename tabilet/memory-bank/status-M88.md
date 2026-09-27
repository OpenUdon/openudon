# Status M88 — Stage 1 step-authoring remediation

**State:** Active. Review gate has not started.

**Provenance:** 2026-09-27 UWS/APItools/OpenUdon stage 1 review R2, R3, R6; each source P2 and local P2, confirmed against clean OpenUdon `c377c8d10e083fabe175b200b1c472f8a1b4dbc8`. Review baseline is the same commit; no uncommitted repository changes were part of the evidence. Published `docs/examples/step-authoring/v1` fixture variants reproduce all three. Historical owner M87 is retired.

| Item | State | Notes |
| --- | --- | --- |
| M88.1 Adopt APItools remediation | `[+]` | Pinned published APItools M78 `26bb05247d6c48f8ee60b9ae178f6ef6d48bbe3d`; compound read/mutation bind refusal and nullable output candidate regressions pass with `GOWORK=off go test ./internal/stepauthoring -count=1`. The v1 wire remains unchanged; current pin truth is updated in `tech-stack.md`. |
| M88.2 Enforce request locations | `[~]` | Correct R2 in `internal/stepauthoring/check.go` and bind/check shared validation; reject wrong body/query/path/header/cookie locations and ambiguous unqualified names before writing. Add generated-UWS regression. |
| M88.3 Validate mapped workflow values | `[ ]` | Correct R3; compare actual mapped input type and requiredness with the contract and source, diagnose unknown expressions honestly, and test rejected/indeterminate binds without mutation. |
| M88.4 Validate inline credential symbols | `[ ]` | Correct R6; reject `credentials.clear`, `credentials.none`, malformed or empty symbols through request mappings and check, with safe auth-alternative regressions. |

## Acceptance and review

All rows and M88 verification in `milestone.md` must pass before a whole-milestone review. Persist the review iteration here before each pass (maximum ten). No gate iteration has started during intake.
