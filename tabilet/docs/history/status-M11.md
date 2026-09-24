# Retired milestone M11 - Status M11 — Public OpenUdon Package Boundary

**Milestone.** M11
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M11.md
**Source status SHA-256.** 11aa50f259dde1c5725754e5efa3762f50c1bdf127275c52b9ddc4d79f2fa649
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
# Status M11 — Public OpenUdon Package Boundary

State of M11 items. See [milestone.md](milestone.md) for milestone scope and
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
| OpenUdon lifecycle migration completed | `[+]` | iCoT/progressive loop, transcript/replay, JSON fallback, artifacts, review handoff validation, package digest, symbolic bindings, credential scanning, and review metadata are local. |
| Final apitools keep boundary preserved | `[+]` | OpenAPI-first API metadata search, discovery, import/lowering, download, local scanning, validation, operation indexing/summaries/auth/ranking, catalog metadata, CLI search/import, and cache support remain in apitools. |
| Non-metadata apitools lifecycle APIs removed downstream | `[+]` | OpenUdon and udon no longer rely on removed lifecycle APIs. |
| Udon consumer migration completed | `[+]` | Udon moved runtime-plan review/handoff helpers and LLM provider plumbing into udon-owned code. |
| OpenUdon static boundary guard added | `[+]` | `openudon check-apitools-boundary` prevents regression to non-metadata apitools lifecycle APIs and private executor imports. |
| Trusted executor handoff split completed | `[+]` | OpenUdon hands UWS Document, OpenAPI files, non-secret run config, and credential binding names to an external executor. |
| Trusted handoff hardening completed | `[+]` | OpenAPI files are digest-covered, symlinked OpenAPI artifacts are rejected, Docker receives declared credential env names only, and invalid operation IDs fail generation. |
| Public-module remediation completed | `[+]` | Public module versions are used, validation fixtures are committed, and public CI runs provider-free module gates. |
| Public-readiness follow-up completed | `[+]` | License, contribution guidance, install docs, direct runner tests, and synthesize package docs are in place. |
| Review-publication cleanup completed | `[+]` | Public docs are self-contained, repository boundary checks run in CI, and MkDocs deploy uses strict build. |
~~~~~~~~~~~~~~~~~~~~
