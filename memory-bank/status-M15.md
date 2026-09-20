# Status M15 — Agentic UWS Authoring For Common SaaS Workflows

State of M15 items. See [milestone.md](milestone.md) for milestone scope and
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
| M15 authoring contract documented | `[+]` | Added public Agentic SaaS Authoring contract for natural-language/guided authoring to reviewed UWS/OpenAPI package. |
| Service priority corpus selected | `[+]` | Selected Slack, Gmail, Jira, HubSpot, Google Drive, Airtable, PagerDuty, Trello, and OpenWeatherMap from n8n/try-n8n plus existing eval evidence. |
| OpenAPI input readiness checked | `[+]` | Documented readiness criteria for local OpenAPI documents, operation IDs, request/response schemas, security schemes, and examples. |
| Credential binding conventions documented | `[+]` | Defined symbolic binding names for selected services without credential values. |
| iCoT prompts sharpened for SaaS authoring | `[+]` | Added OpenUdon-native SaaS guidance for operation choice, mappings, side effects, and n8n evidence handling. |
| Request mapping guidance improved | `[+]` | Strengthened prompt and repair guidance for required request field mapping. |
| Data-flow review evidence improved | `[+]` | Added SaaS data-flow expectations for request sources, prior-step bindings, output paths, and side-effect scope. |
| Quality repair hints improved | `[+]` | Added targeted CLI next-action hints for OpenAPI readiness, intent operation selection, required params, response paths, and credential schemes. |
| Golden authoring fixtures added | `[+]` | Added `reference/authoring.json` to the OpenUdon-native Slack message audit-log strict fixture. |
| n8n evidence references preserved | `[+]` | Public docs preserve n8n-derived fixture provenance as service-priority/mapping evidence while deferring import. |
| Documentation and tutorial added | `[+]` | Added public Agentic SaaS Authoring docs and linked them from README, authoring docs, and MkDocs nav. |
