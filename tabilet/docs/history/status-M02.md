# Retired milestone M02 - Status M02 — Eval Corpus And Reference Discipline

**Milestone.** M02
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M02.md
**Source status SHA-256.** 1a98144c6aa1e94fbb42d750d3ae25f73f5c0267d07708572bd74c2ab8a01e66
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
# Status M02 — Eval Corpus And Reference Discipline

State of M02 items. See [milestone.md](milestone.md) for milestone scope and
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
| Eval harness implemented | `[+]` | Eval runs generate JSON/Markdown reports with pass/fail status, comparison data, provider/model metadata, and ignored run artifacts. |
| Expanded eval corpus implemented | `[+]` | Curated examples cover OpenAPI auth, pagination, request bodies, response extraction, writes, multi-service chains, runtime-only functions, approved/denied runtimes, and negative policy cases. |
| Reference policy support implemented | `[+]` | Per-fixture reference policies classify advisory, warning, and blocking concerns. |
| n8n reducibility fixtures added | `[+]` | Airtable, Gmail, Drive, HubSpot, Jira, OpenWeatherMap, PagerDuty, Slack, and Trello advisory fixtures prove OpenAPI-backed reducibility without n8n runtime semantics. |
| Release eval criteria implemented | `[+]` | Release eval gates check pass rate, structured mode, attempts, blocking reference issues, and secret-scan failures. |
| Regression reporting implemented | `[+]` | Eval reports distinguish behavioral regressions from acceptable naming or review-text drift. |
~~~~~~~~~~~~~~~~~~~~
