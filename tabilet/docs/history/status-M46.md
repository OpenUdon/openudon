# Retired milestone M46 - M46 - UWS 1.4 Source-Family Eval Coverage

**Milestone.** M46
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M46.md
**Source status SHA-256.** f3bf81cda2771e295e715066ac22b98d2e1e607aaefe812c7c19109611d73ccc
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
# M46 - UWS 1.4 Source-Family Eval Coverage

## Status

Complete.

## Task State

| Item | State | Notes |
|---|---|---|
| Source directory coverage | `[+]` | Package inventory, review handoff, trusted-runner staging, and iCoT seed copy include `graphql/`, `openrpc/`, `grpc-protobuf/`, and `odata/`. |
| Synthesis and quality coverage | `[+]` | Local source registries, operation lookup, request mapping, quality checks, and UWS version selection consume apitools summaries for GraphQL, OpenRPC, gRPC/protobuf, and OData. |
| UWS 1.4 schema support | `[+]` | Embedded schema set includes `1.4.0`; UWS 1.4 source-bound workflows validate locally. |
| Eval fixtures | `[+]` | Added one M68-backed fixture each for GraphQL, OpenRPC, gRPC/protobuf, and OData under `examples/eval/`. |
| iCoT local discovery | `[+]` | iCoT discovers and ranks operations from the four new source-aligned directories. |
| Documentation and memory | `[+]` | Public docs and memory bank now describe UWS 1.4 source-family package/generation coverage while preserving trusted-executor execution boundaries. |

## Verification

- `go test ./internal/packageartifacts ./internal/synthesize ./internal/icot/elicitor ./internal/uwsschema`
- `go run ./cmd/openudon build --example examples/eval/graphql-starwars-hero`
- `go run ./cmd/openudon build --example examples/eval/openrpc-simple-math-addition`
- `go run ./cmd/openudon build --example examples/eval/grpc-trace-export`
- `go run ./cmd/openudon build --example examples/eval/odata-customers-query`
- `go test ./...`
- `GOWORK=off go test ./...`
- `go vet ./...`
- `go run ./cmd/openudon check`
- `go run ./cmd/openudon check-apitools-boundary`
- `go run ./cmd/openudon check-doc-memory`
- `go run ./cmd/openudon validate ./examples/uws-validation`
- `go run ./cmd/icot variants validate --root examples/eval`
- `go run ./cmd/icot variants coverage --root examples/eval`
- `make eval-seed-build`
- `git diff --check`
- `git -C ../tofu diff --check -- openudon/memory-bank openudon/evolution`

## Notes

- The fixtures reuse the reviewed M68 source corpus from `../apitools/catalog-openapi-cache`.
- OpenUdon still does not implement GraphQL, JSON-RPC, gRPC, or OData protocol execution; approved
  execution remains delegated to a trusted runtime.
- Evolution V5 records this OpenUdon source-family coverage boundary. Further evolution is only
  needed if the trusted executor contract or public UWS semantics change.
~~~~~~~~~~~~~~~~~~~~
