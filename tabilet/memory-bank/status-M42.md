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
