# Status M01 — Post-POC Baseline

State of M01 items. See [milestone.md](milestone.md) for milestone scope and
acceptance criteria.

Status markers:

| Symbol | Suggested Status | Interpretation |
|---|---|---|
| `[ ]` | Pending | Item not started or pending action. |
| `[+]` | Completed | Item finished or done. |
| `[~]` | In Progress | Item is being worked on. |
| `[!]` | Blocked | Item requires attention or is on hold. |
| `[X]` | Cancelled | Item is no longer needed. |

| Item | State | Notes |
|---|---|---|
| Thin OpenUdon CLI baseline implemented | `[+]` | `cmd/openudon` supports local checks, synthesis, build, promote, assess, eval, readiness, approval template, and trusted run commands. |
| Guided iCoT CLI baseline implemented | `[+]` | `cmd/icot` authors `project.md` and `workflows/intent.hcl` without executing workflows. |
| Authoring artifact model implemented | `[+]` | Project brief plus structured `workflows/intent.hcl` are the source artifacts. |
| Deterministic generation baseline implemented | `[+]` | Workflow HCL, UWS YAML, plans, discovery, refinement, review, handoff, and quality reports are generated deterministically. |
| Refinement loop baseline implemented | `[+]` | Bounded repair attempts are recorded in `expected/refinement.json`. |
| Deterministic quality gates baseline implemented | `[+]` | Project policy, OpenAPI availability, intent, data flow, workflow compilation, plan matching, UWS validation, review evidence, handoff policy, credentials, side effects, and secret scanning are checked. |
| Deterministic versus provider-backed checks documented | `[+]` | Normal development stays provider-free; real-provider evidence remains local/manual. |
