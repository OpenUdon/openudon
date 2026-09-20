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
