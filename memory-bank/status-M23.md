# Status M23 — SaaS Operator Release Readiness

State of M23 items. See [milestone.md](milestone.md) for milestone scope and
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
| Operator workflow documented | `[+]` | Added the SaaS operator release path with ignored-workdir author/build/assess/review/approval-template/dry-run/archive guidance. |
| Demo path selected | `[+]` | Selected `gmail-send-audit-receipt` as the single-service demo and `order-fulfillment-chain` as the multi-service demo. |
| Boundary language tightened | `[+]` | Clarified OpenUdon versus n8n, live SaaS providers, desired-state engines, external orchestration, and udon in operator, release, review, and related-project docs. |
| Release checklist updated | `[+]` | Added SaaS deterministic evidence, n8n bridge validation, selected fixture lint, dry-run demo evidence, and optional real-provider/provider-drift expectations. |
| Tutorial coverage updated | `[+]` | Added the order fulfillment tutorial and linked Gmail release evidence back to the provider-free operator path. |
| Provider drift posture reviewed | `[+]` | Release stewardship and release-note template now separate deterministic gates from optional real-provider drift evidence and rerun notes. |
| Public docs checked | `[+]` | Added MkDocs nav entries for the SaaS operator release path and order fulfillment tutorial; strict MkDocs build passed. |
| Deterministic gates checked | `[+]` | Ran tests, vet, make check, release-check, strict docs, doc-memory, selected lint, UWS validation, n8n bridge validation, trusted dry-run smoke, and diff checks. |
