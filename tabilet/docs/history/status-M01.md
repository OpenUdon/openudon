# Retired milestone M01 - Status M01 — Post-POC Baseline

**Milestone.** M01
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M01.md
**Source status SHA-256.** 0c5d89654616e92b09b7d875e2a487d1d0f0866b64158f6bab5a324f8611b099
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
# Status M01 — Post-POC Baseline

State of M01 items. See [milestone.md](milestone.md) for milestone scope and
acceptance criteria.

Status markers:

| Symbol | Suggested Status | Interpretation |
|---|---|---|
| `[ ]` | Pending | Item not started or pending action. |
| `[+]` | Completed | Item finished or done. |
| `[~]` | In Progress | Item is being worked on. |
| `[!]` | Blocked | Item requires attention or is on hold. |
| `[X]` | Cancelled | Item is no longer needed. |

| Item | State | Notes |
|---|---|---|
| Thin OpenUdon CLI baseline implemented | `[+]` | `cmd/openudon` supports local checks, synthesis, build, promote, assess, eval, readiness, approval template, and trusted run commands. |
| Guided iCoT CLI baseline implemented | `[+]` | `cmd/icot` authors `project.md` and `workflows/intent.hcl` without executing workflows. |
| Authoring artifact model implemented | `[+]` | Project brief plus structured `workflows/intent.hcl` are the source artifacts. |
| Deterministic generation baseline implemented | `[+]` | Workflow HCL, UWS YAML, plans, discovery, refinement, review, handoff, and quality reports are generated deterministically. |
| Refinement loop baseline implemented | `[+]` | Bounded repair attempts are recorded in `expected/refinement.json`. |
| Deterministic quality gates baseline implemented | `[+]` | Project policy, OpenAPI availability, intent, data flow, workflow compilation, plan matching, UWS validation, review evidence, handoff policy, credentials, side effects, and secret scanning are checked. |
| Deterministic versus provider-backed checks documented | `[+]` | Normal development stays provider-free; real-provider evidence remains local/manual. |
~~~~~~~~~~~~~~~~~~~~
