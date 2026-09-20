# Status M20 — End-To-End SaaS Authoring Trials

State of M20 items. See [milestone.md](milestone.md) for milestone scope and
acceptance criteria.

Status markers:

| Symbol | Suggested Status | Interpretation |
|---|---|---|
| `[ ]` | Pending | Item not started or pending action. |
| `[+]` | Completed | Item finished or done. |
| `[~]` | In Progress | Item is being worked on. |
| `[!]` | Blocked | Item requires attention or is on hold. |
| `[X]` | Cancelled | Item is no longer needed. |

Rows may be implemented together when the changes are tightly coupled. Verify
the combined change before committing and name every covered row in the
handoff.

| Item | State | Notes |
|---|---|---|
| Trial brief set selected | `[+]` | Selected Slack, Gmail, Slack/Jira, Jira/Slack/Drive, synthetic order chain, HubSpot, Airtable, and OpenWeatherMap trials. |
| Trial harness defined | `[+]` | Documented ignored `.openudon-run/m20-trials` harness for lint, build, approval-template, and `openudon run --dry-run`. |
| Guided authoring trials run | `[+]` | Ran `cmd/icot lint` for all eight selected briefs; all passed authoring lint. |
| Package quality trials run | `[+]` | Ran build/assess package trials from reference intents; Slack, Gmail, and weather converge, while multi-service/security and advisory OpenAPI mapping gaps are recorded. |
| Trusted dry-run trials run | `[+]` | Validated approval-template and sandbox `openudon run --dry-run` for Slack, Gmail, and weather package trials. |
| Gap matrix written | `[+]` | Added M20 gap matrix covering function contract wording, OpenAPI security mapping, copied OpenAPI parameter shape, review/handoff, and live-provider deferral. |
| Promotion candidates chosen | `[+]` | Chose Slack, Gmail, and weather as immediate M21 candidates; multi-service strict fixtures are repair-first; HubSpot/Airtable remain advisory until normalized. |
| Public or memory docs updated | `[+]` | Added public SaaS Authoring Trials docs and linked them from corpus, eval gallery, and MkDocs nav. |
| Deterministic gates checked | `[+]` | Ran tests, vet, make check, release-check, strict MkDocs, doc-memory, selected lint, UWS validation, trusted dry-run trials, and diff checks. |
