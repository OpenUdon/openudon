# Retired milestone M09 - Status M09 — Complete OpenUdon Package Integration

**Milestone.** M09
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M09.md
**Source status SHA-256.** c7a246b37dd467863592c2b5d02f6e20892795171c3a133e04ea58289f26f7dc
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
# Status M09 — Complete OpenUdon Package Integration

State of M09 items. See [milestone.md](milestone.md) for milestone scope and
acceptance criteria.

Status markers:

| Symbol | Suggested Status | Interpretation |
|---|---|---|
| `[ ]` | Pending | Item not started or pending action. |
| `[+]` | Completed | Item finished or done. |
| `[~]` | In Progress | Item is being worked on. |
| `[!]` | Blocked | Item requires attention or is on hold. |
| `[X]` | Cancelled | Item is no longer needed. |

Each table row is a commit unit once implementation begins: after flipping a
row to `[+]`, verify the change, update the memory bank/docs, and make a scoped
commit before starting the next row. If multiple rows are inseparable, use one
coherent commit and name every covered row in the handoff.

| Item | State | Notes |
|---|---|---|
| `workflows/workflow.hcl` integration implemented | `[+]` | `tfconvert` now calls OpenUdon's deterministic package-from-intent path to build normal workflow HCL from generated intent |
| `workflows/workflow.uws.yaml` integration implemented | `[+]` | Converted packages are promoted through the normal UWS export and schema validation path |
| Expected plan artifacts emitted | `[+]` | Converted packages emit `expected/plan.json` and `expected/plan.md` through the shared synthesize plan writer |
| Discovery artifacts emitted | `[+]` | OpenAPI inputs are staged under package-local `openapi/` and normal discovery evidence is emitted |
| Review artifacts integrated | `[+]` | Normal OpenUdon review evidence replaces the draft-only review note after package construction |
| Quality artifacts integrated | `[+]` | Converted packages emit normal `expected/quality.json` and `expected/quality.md` reports |
| Handoff manifest integration implemented | `[+]` | Normal review handoff manifests are generated and validated for converted candidates |
| Package digest behavior reused | `[+]` | Integration coverage computes the normal review handoff digest over converted package inputs |
| Converted artifacts remain unapproved by default | `[+]` | Handoff manifests keep the generated state and require the normal approval state machine before trusted execution |
| Unresolved TODO quality behavior implemented | `[+]` | Unresolved operation TODOs remain reviewable artifacts while strict mode and quality gates fail them |
| Unsafe assumption quality behavior implemented | `[+]` | Conversion diagnostics with strict-failure semantics are consumed by normal quality checks instead of being hidden |
| Full package integration tests added | `[+]` | Converter integration tests cover normal artifacts, quality failure, generated approval state, digest calculation, and handoff boundaries |
~~~~~~~~~~~~~~~~~~~~
