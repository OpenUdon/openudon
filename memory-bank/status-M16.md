# Status M16 — SaaS Authoring Corpus Hardening

State of M16 items. See [milestone.md](milestone.md) for milestone scope and
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
| Corpus inventory audited | `[+]` | Documented selected-service native and n8n-derived fixture evidence in `docs/saas-authoring-corpus.md`. |
| Fixture policy classified | `[+]` | Classified Slack, Gmail, and Slack/Jira as strict native coverage; kept n8n-derived fixtures advisory. |
| Graduation criteria documented | `[+]` | Added advisory-to-strict graduation criteria for project brief, OpenAPI, intent, credentials, side effects, policy, and authoring metadata. |
| OpenAPI readiness matrix added | `[+]` | Added readiness matrix covering operation IDs, request fields, response fields, security evidence, and known gaps. |
| Credential binding conventions aligned | `[+]` | Aligned documented symbolic names with fixtures, including `openweathermap_appid` for OpenWeatherMap. |
| Request mapping references hardened | `[+]` | Added strict-native authoring metadata for Slack, Gmail, and Slack/Jira request mappings. |
| Response path references hardened | `[+]` | Added strict-native authoring metadata for consumed response paths in Slack, Gmail, and Slack/Jira fixtures. |
| Native golden metadata expanded | `[+]` | Added `reference/authoring.json` to Gmail and Slack/Jira; added fixture policy metadata to Slack. |
| Initial strict SaaS golden set chosen | `[+]` | Chose Slack message audit log, Gmail send audit receipt, and Slack-to-Jira issue intake as the initial strict set. |
| Quality/eval gates checked | `[+]` | Verified deterministic tests, vet, make check, release-check, UWS validation, strict MkDocs build, doc-memory check, JSON parsing, and diff checks. |
| Public docs updated | `[+]` | Added SaaS Authoring Corpus docs and linked them from Agentic SaaS Authoring, Eval Gallery, and MkDocs nav. |
