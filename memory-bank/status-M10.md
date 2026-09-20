# Status M10 — Cross-Repo Dependency Stewardship

State of M10 items. See [milestone.md](milestone.md) for milestone scope and
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
| UWS ownership boundary maintained | `[+]` | Public workflow semantics remain in `../uws`; OpenUdon consumes public UWS contracts. |
| apitools metadata boundary stewarded | `[+]` | OpenUdon uses `apitools` only for OpenAPI-first API metadata discovery/import/lowering/search/indexing/summaries/ranking; native Discovery and Smithy handling stays upstream. |
| udon execution boundary maintained | `[+]` | OpenUdon does not import udon Go packages and invokes executors only through trusted CLI/Docker handoff. |
| External orchestration boundary maintained | `[+]` | Managed orchestration remains outside OpenUdon; OpenUdon owns local package evidence and trusted-runner enforcement. |
| Readiness reporting implemented | `[+]` | Local readiness reports track sibling checkout state, deterministic gates, git state, ignored artifacts, and provider env presence booleans. |
| Closed XRD matrix maintained | `[+]` | Cross-repo dependency risks are documented as regression responsibilities rather than absorbed into OpenUdon. |
