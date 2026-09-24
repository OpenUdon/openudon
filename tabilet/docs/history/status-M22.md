# Retired milestone M22 - Status M22 — n8n Pattern Bridge

**Milestone.** M22
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M22.md
**Source status SHA-256.** 9a7f2d8ddf20e104200f972d96096cd0be54421a56383b8e7754c69938888725
**Source milestone snapshot.** tabilet/docs/history/milestone-before-legacy-retirement.md.txt
**Snapshot SHA-256.** 26884eeda9af4ded30e6d545a33dca84c401d2320c22355fe4fb0336d5dcac71
**Evidence.** 71a4f78afbcf2180fc478ffa89c53544c9160648
**Worktree.** includes uncommitted changes
**Review.** not established
**Review iterations.** not recorded
**Verification.** Original status bytes and full earlier milestone bytes preserved by SHA-256; no fresh acceptance claim.
**Consolidated into.** Current milestone dashboard and maintained memory-bank guidance; full earlier text remains in the frozen snapshot.

## Status record

~~~~~~~~~~~~~~~~~~~~markdown
# Status M22 — n8n Pattern Bridge

State of M22 items. See [milestone.md](milestone.md) for milestone scope and
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
| n8n evidence contract designed | `[+]` | Added `openudon.n8n-pattern-summary.v1` summaries with services, nodes, operations, OpenAPI candidates, credentials, data-flow hints, unsupported semantics, generated candidate paths, and validation status. |
| Unsupported semantics cataloged | `[+]` | Documented trigger, schedule/wait, expression, item batching/pagination, binary data, custom code, and credential handling as diagnostics, TODOs, manual contracts, or unsupported behavior. |
| Source evidence selected | `[+]` | Selected `n8n-slack-message-post`, `n8n-google-drive-file-upload`, and `n8n-hubspot-deal-list` as representative bridge summaries using fixture-local provenance plus `../try-n8n` matrix evidence. |
| Bridge output boundary documented | `[+]` | Public docs and summary files state the bridge is authoring assistance only, not executable import, n8n runtime emulation, provider execution, or UWS translation. |
| Optional local harness implemented | `[+]` | Added deterministic `openudon n8n-bridge validate` for one summary file or every `reference/n8n-bridge.json` under the eval root. |
| Generated candidates validated | `[+]` | Summaries point to existing candidate `project.md` and `reference/intent.hcl` files; selected fixture lint remains the promotion gate, with HubSpot explicitly blocked until request-field/pagination evidence is repaired. |
| Public docs updated | `[+]` | Added n8n pattern bridge docs and linked the bridge from authoring, agentic SaaS authoring, SaaS corpus, and eval gallery docs. |
| Deterministic gates checked | `[+]` | Ran tests, vet, make check, release-check, strict docs, doc-memory, UWS validation, n8n bridge validation, selected n8n fixture lint, and diff checks. |
~~~~~~~~~~~~~~~~~~~~
