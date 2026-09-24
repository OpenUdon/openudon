# Retired milestone M44 - M44 iCoT, Examples, And Eval Coverage

**Milestone.** M44
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M44.md
**Source status SHA-256.** 917fc7fd1d352d42e8c1501cc74299b29130eeee49f08a920aefc5794ae42720
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
# M44 iCoT, Examples, And Eval Coverage

## Status

Complete.

## Task State

| Item | State | Notes |
|---|---|---|
| iCoT local source discovery | `[+]` | iCoT scans `asyncapi/` and includes operation candidates with provenance `asyncapi`. |
| Deterministic AsyncAPI fixture | `[+]` | Added `examples/eval/asyncapi-billing-event` with package-local source, intent, generated workflow artifacts, plan, review, quality, and handoff. |
| Representative AsyncAPI artifact fixture | `[+]` | Added `examples/eval/asyncapi-streetlights-mqtt` with the tracked Streetlights MQTT AsyncAPI artifact vendored from `../apitools`, strict `dimLight` intent mappings, generated package artifacts, and AsyncAPI-only build coverage. |
| Authoring variants | `[+]` | Added positive, missing-detail, and unsafe review-bypass variants for the AsyncAPI fixture. |
| Provider execution exclusion | `[+]` | Fixture is provider-free and documents trusted-runtime ownership for real event delivery. |

## Verification

- `openudon build --example examples/eval/asyncapi-billing-event`
- Provider-free iCoT scorecard includes `asyncapi-billing-event` and its authoring variants.
- `go run ./cmd/openudon build --example examples/eval/asyncapi-streetlights-mqtt`
- `go run ./cmd/openudon validate examples/eval/asyncapi-streetlights-mqtt/workflows/workflow.uws.yaml`
- `go run ./cmd/icot lint --example examples/eval/asyncapi-streetlights-mqtt`
- `go run ./cmd/icot variants validate --root examples/eval --name asyncapi-streetlights-mqtt`
~~~~~~~~~~~~~~~~~~~~
