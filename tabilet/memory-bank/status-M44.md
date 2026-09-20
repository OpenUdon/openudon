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
