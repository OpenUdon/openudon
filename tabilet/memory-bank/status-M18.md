# Status M18 — Multi-Service SaaS Workflow Patterns

State of M18 items. See [milestone.md](milestone.md) for milestone scope and
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
| Pattern inventory selected | `[+]` | Documented lookup-then-notify, ticket-then-message, send-then-audit, upload/archive, pagination, webhook send, and lookup-then-create patterns. |
| Multi-service fixture gaps audited | `[+]` | Compared IT Ops, order fulfillment, Gmail/Slack audit, pagination, webhook, and alert fixtures in `docs/multi-service-saas-patterns.md`. |
| Cross-service data-flow mappings hardened | `[+]` | Added `reference/authoring.json` mappings for incident response archive and order fulfillment strict fixtures. |
| Credential scope evidence improved | `[+]` | Documented distinct per-service bindings for Slack/Jira, Jira/Slack/Drive, and Customer/Inventory/Order fixtures. |
| Side-effect policy coverage improved | `[+]` | Documented create/post/upload/send side-effect posture and kept execution behind review/trusted-runner approval. |
| Native strict multi-service set chosen | `[+]` | Chose Slack-to-Jira issue intake, incident response archive, and order fulfillment chain as strict multi-service set. |
| Quality gates checked | `[+]` | Ran deterministic tests, release checks, strict MkDocs build, UWS validation, doc-memory check, and selected iCoT lint for strict multi-service fixtures. |
| Public docs updated | `[+]` | Added Multi-Service SaaS Patterns docs and linked them from corpus, eval gallery, and MkDocs nav. |
