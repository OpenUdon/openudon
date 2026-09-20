# Status M24 — Release Gate Consolidation And Evidence Automation

State of M24 items. See [milestone.md](milestone.md) for milestone scope and
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
| M24 scope documented | `[+]` | Added the release gate consolidation milestone and status tracking. |
| SaaS release gate target added | `[+]` | Added provider-free `make release-saas-check` for release-check, UWS validation, doc-memory, n8n bridge validation, strict MkDocs, selected fixture lint, and Gmail/order fulfillment dry-run demos. |
| Release docs aligned | `[+]` | Updated release stewardship, SaaS operator release path, release-note template, and README to name `make release-saas-check` as the comprehensive local SaaS release gate. |
| Release checklist tests updated | `[+]` | Added doc/test coverage that the Makefile and release docs expose the new local gate. |
| Deterministic gates checked | `[+]` | Ran `make release-saas-check`, which covered tests, vet, make check, UWS validation, doc-memory, n8n bridge validation, strict MkDocs, selected fixture lint, trusted dry-run demos, and diff checks. |
