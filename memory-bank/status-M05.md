# Status M05 — Workflow Artifact Power

State of M05 items. See [milestone.md](milestone.md) for milestone scope and
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
| Switch branch support implemented | `[+]` | Prompt, plan, review, workflow, UWS, and quality coverage preserve switch branches. |
| Loop support implemented | `[+]` | Loop artifacts are preserved through intent, workflow HCL, UWS export, plan, review, and quality checks. |
| Structural result support implemented | `[+]` | Generated structural outputs are validated against expected plans and UWS artifacts. |
| Failure branch policy implemented | `[+]` | Failure branches are allowed only when brief or intent explicitly asks for failure routing. |
| Retry policy constrained | `[+]` | Retries are allowed only when explicitly requested; side-effectful retries require retry/idempotency policy. |
| Timeout and idempotency opt-in preserved | `[+]` | UWS 1.1 timeout and workflow idempotency metadata are preserved when explicitly requested. |
| Runtime/profile eval coverage added | `[+]` | Approved `fnct`, approved `cmd`, denied command/SSH, and future profile-boundary fixtures are covered. |
