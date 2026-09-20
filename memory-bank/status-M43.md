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
