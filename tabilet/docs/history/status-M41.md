# Retired milestone M41 - M41 UWS 1.3 Contract Baseline

**Milestone.** M41
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M41.md
**Source status SHA-256.** d87dedd1d0cc53884e119c18a8e4ef6862737b4aa86e47e77ceeee34513aca23
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
# M41 UWS 1.3 Contract Baseline

## Status

Complete.

## Task State

| Item | State | Notes |
|---|---|---|
| Consume UWS 1.3 model/schema revision | `[+]` | Published UWS `68106ab` exposes `versions/1.3.0.json`, `SourceDescriptionTypeAsyncAPI`, and validator support; OpenUdon consumes the public pseudo-version. |
| Embed UWS 1.3 schema | `[+]` | Added `internal/uwsschema/versions/1.3.0.json` from the sibling UWS snapshot. |
| Schema lookup coverage | `[+]` | Embedded schema tests include `1.3.0`. |
| Validate UWS 1.3 AsyncAPI document | `[+]` | Added `uwsvalidate` and `openudon validate` coverage for `sourceDescriptions[].type: asyncapi`. |

## Verification

- `go test ./cmd/openudon ./internal/uwsschema ./internal/uwsvalidate ./internal/packageartifacts ./internal/synthesize ./internal/icot/elicitor`
- `GOWORK=off go test ./internal/synthesize ./internal/uwsexec ./internal/uwsvalidate ./cmd/openudon`
~~~~~~~~~~~~~~~~~~~~
