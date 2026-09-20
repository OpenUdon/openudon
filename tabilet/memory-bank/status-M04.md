# Status M04 — Quality Gate Hardening

State of M04 items. See [milestone.md](milestone.md) for milestone scope and
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
| Data-flow quality checks hardened | `[+]` | Checks cover missing dependencies, ambiguous sources, invalid response paths, and undeclared function inputs. |
| Credential quality checks hardened | `[+]` | Checks cover binding declarations, OpenAPI security schemes, request placement, and secret-value leakage. |
| Side-effect quality checks hardened | `[+]` | Checks cover write operations, customer communications, command/SSH runtimes, and production endpoint language. |
| Review evidence gates hardened | `[+]` | Evidence is required for side effects, unresolved risks, skipped execution, credential binding names, approval states, sandbox proof runs, and trusted-runner handoff. |
| Artifact versus infrastructure failure kind added | `[+]` | Synthesis quality checks carry `failure_kind` to distinguish artifact failures from infrastructure failures. |
| Repair guidance implemented | `[+]` | Quality failures include concrete next-action guidance. |
