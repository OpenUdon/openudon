# Retired milestone B01 - Status B01 — Project Scaffold And Direction

**Milestone.** B01
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-B01.md
**Source status SHA-256.** 78405c060ecd1b0c867f4b1d02acfb6075811bb905881940745f5c784a15c320
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
# Status B01 — Project Scaffold And Direction

State of B01 items. B01 captures the pre-numbered project setup that happened
before the formal milestone list in [milestone.md](milestone.md).

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
| Initial product direction documented | `[+]` | OpenUdon is defined as public UWS authoring, review, package, and executor-handoff tooling. |
| Memory bank scaffold created | `[+]` | `product.md`, `architecture.md`, `tech-stack.md`, and `milestone.md` established as active project memory; per-milestone task history now lives in `status-Mx.md` files. |
| Evolution v1 scaffold created | `[+]` | `evolution/prompt-v1.md` and `evolution/result-v1.md` established for direction snapshots. |
| Go module established | `[+]` | Module path is `github.com/OpenUdon/openudon`. |
| Optional sibling workspace layout established | `[+]` | Local sibling development happens through parent `go.work`, not committed `replace ../...` directives. |
| Source ownership boundaries documented | `[+]` | UWS, apitools, Ramen, udon, external orchestration, and OpenUdon ownership boundaries recorded in memory bank and AGENTS guidance. |
~~~~~~~~~~~~~~~~~~~~
