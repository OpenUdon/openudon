# Retired milestone M21 - Status M21 — Strict SaaS Corpus Expansion

**Milestone.** M21
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M21.md
**Source status SHA-256.** e8a6ece7a420838fd6b32471b46a2a492a5f431a4306e20b6732df6c07e7d4ea
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
~~~~~~~~~~~~~~~~~~~~
