# Status M17 — Guided SaaS Authoring UX

State of M17 items. See [milestone.md](milestone.md) for milestone scope and
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
| Guided question audit | `[+]` | Reviewed progressive iCoT readiness questions against the M16 strict native corpus. |
| Service selection prompts improved | `[+]` | Updated OpenAPI document and operation prompts to ask for listed local operation IDs and unresolved capability gaps. |
| Credential binding prompts improved | `[+]` | Updated prompts/readiness to ask for symbolic credential binding names only, never secret values. |
| Request mapping prompts improved | `[+]` | Updated required-field prompts to ask for input, safe literal, prior-step output, or credential binding sources. |
| Response/output prompts improved | `[+]` | Updated output prompts/readiness to ask for known response paths or function outputs without guessing provider fields. |
| Side-effect prompts improved | `[+]` | Added guided distinction between read-only, sandbox-only, and after-approval execution, with read-only defaulting for read/fetch/list-style drafts. |
| Repair feedback tightened | `[+]` | Added operation-choice hints and more concrete readiness messages for OpenAPI, operation, field, credential, output, and safety gaps. |
| Guided authoring fixtures checked | `[+]` | Ran `icot lint` on Slack message audit log, Gmail send audit receipt, and Slack-to-Jira issue intake. |
| Public docs updated | `[+]` | Updated iCoT and authoring docs with M17 guided SaaS UX behavior and limits. |
