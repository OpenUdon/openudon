# Status M21 — Strict SaaS Corpus Expansion

State of M21 items. See [milestone.md](milestone.md) for milestone scope and
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
| M20 promotion candidates reviewed | `[+]` | Promoted Slack, Gmail, weather, Slack/Jira intake, incident archive, and order fulfillment as strict package/dry-run coverage; kept HubSpot/Airtable advisory. |
| Local OpenAPI slices hardened | `[+]` | Hardened request placement for bearer security so strict multi-service OpenAPI slices expose `Authorization` consistently during workflow generation. |
| Reference intents added or hardened | `[+]` | Verified promoted reference intents build from checked-in `reference/intent.hcl`; no intent shape changes were needed. |
| Authoring metadata added | `[+]` | Added `reference/authoring.json` for `weather-toronto` with request mappings, response paths, and credential binding evidence. |
| Fixture policies updated | `[+]` | Updated `weather-toronto` strict policy wording; HubSpot and Airtable remain advisory until copied OpenAPI request mappings are normalized. |
| Corpus docs updated | `[+]` | Updated SaaS corpus, trials, eval gallery, and multi-service pattern docs with M21 promotion and bearer-security repair evidence. |
| Quality/eval tests updated | `[+]` | Added bearer Authorization placement coverage and eval fixture assertions for strict weather authoring metadata and bindings. |
| Deterministic gates checked | `[+]` | Ran tests, vet, make check, release-check, strict docs, doc-memory, selected lint, UWS validation, promoted fixture dry-runs, and diff checks. |
