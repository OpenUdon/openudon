# Retired milestone M42 - M42 AsyncAPI Metadata Boundary

**Milestone.** M42
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M42.md
**Source status SHA-256.** 921bd7b2ab7d42a4afe95f0641d97e450bd3851053632a0f49742585be1a9498
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
# M42 AsyncAPI Metadata Boundary

## Status

Complete.

## Task State

| Item | State | Notes |
|---|---|---|
| apitools protocol classification | `[+]` | Published apitools `1eba4ca` maps AsyncAPI to protocol `asyncapi`, UWS source type `asyncapi`, and source-aligned artifact directory `asyncapi/`. |
| Local AsyncAPI file metadata | `[+]` | OpenUdon scans `asyncapi/*.json`, `*.yaml`, and `*.yml`, reads root `asyncapi`, `info`, and root `operations` keys, and exposes operation summaries with provenance `asyncapi`. |
| Conservative schema interpretation | `[+]` | Payload/header schemas are opaque; OpenUdon requires explicit request mapping locations instead of inferring AsyncAPI fields. |
| Browser profile exclusion | `[+]` | No browser-profile catalog or source support was added. |

## Verification

- Focused iCoT discovery test covers package-local AsyncAPI operation candidates.
- apitools catalog tests cover AsyncAPI protocol classification, UWS source type, and artifact directory mapping.
~~~~~~~~~~~~~~~~~~~~
