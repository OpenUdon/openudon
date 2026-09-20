# Status M02 — Eval Corpus And Reference Discipline

State of M02 items. See [milestone.md](milestone.md) for milestone scope and
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
| Eval harness implemented | `[+]` | Eval runs generate JSON/Markdown reports with pass/fail status, comparison data, provider/model metadata, and ignored run artifacts. |
| Expanded eval corpus implemented | `[+]` | Curated examples cover OpenAPI auth, pagination, request bodies, response extraction, writes, multi-service chains, runtime-only functions, approved/denied runtimes, and negative policy cases. |
| Reference policy support implemented | `[+]` | Per-fixture reference policies classify advisory, warning, and blocking concerns. |
| n8n reducibility fixtures added | `[+]` | Airtable, Gmail, Drive, HubSpot, Jira, OpenWeatherMap, PagerDuty, Slack, and Trello advisory fixtures prove OpenAPI-backed reducibility without n8n runtime semantics. |
| Release eval criteria implemented | `[+]` | Release eval gates check pass rate, structured mode, attempts, blocking reference issues, and secret-scan failures. |
| Regression reporting implemented | `[+]` | Eval reports distinguish behavioral regressions from acceptable naming or review-text drift. |
