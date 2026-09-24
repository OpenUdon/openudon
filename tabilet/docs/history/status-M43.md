# Retired milestone M43 - M43 OpenUdon Core Package And Generation Support

**Milestone.** M43
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M43.md
**Source status SHA-256.** 128d8f132d8d5380b1f8e76a3f47ea95b16a62b4d94bc9eca3538fb9dbd0b7a5
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
# M43 OpenUdon Core Package And Generation Support

## Status

Complete.

## Task State

| Item | State | Notes |
|---|---|---|
| Package artifact collection | `[+]` | `asyncapi/` files and associated advisory sidecars participate in required package paths and handoff inventory. |
| Source type inference and sniffing | `[+]` | `asyncapi/` paths and root `asyncapi` documents map to UWS source type `asyncapi`. |
| UWS version selection | `[+]` | AsyncAPI intent sources emit `uws: 1.3.0`. |
| Generic selectors | `[+]` | AsyncAPI operation names emit `sourceOperationId`; `#/...` operations emit `sourceOperationRef`. |
| Intent compatibility | `[+]` | Existing `http`/`openapi` step types remain the source-bound aliases; no new intent step type was added. |
| Explicit request mappings | `[+]` | AsyncAPI field metadata is opaque, so unqualified request mappings fail and explicit `body.*`/`header.*` mappings are preserved. |

## Verification

- Added focused generation tests for UWS 1.3 AsyncAPI, operation refs, and explicit request placement.
- Built `examples/eval/asyncapi-billing-event` successfully.
- OpenUdon now uses the published `uws1.SourceDescriptionTypeAsyncAPI` symbol instead of a local compatibility constant.
~~~~~~~~~~~~~~~~~~~~
