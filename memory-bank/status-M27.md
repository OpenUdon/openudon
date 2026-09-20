# M27 — UWS 1.2 First-Class API Source Types

## Goal

Adopt UWS 1.2 typed API source descriptions across OpenUdon once `../uws`, `../udon`, and
`../apitools` can describe, execute, and materialize OpenAPI, Google Discovery, and AWS Smithy
sources without treating every executable API document as OpenAPI.

## Status

| Item | State | Notes |
|---|---|---|
| Track UWS 1.2 public contract | `[+]` | OpenUdon consumes UWS 1.2 schema artifacts and emits typed source descriptions only when a non-OpenAPI source or generic selector requires it. |
| Add generic intent source alias | `[+]` | `source = "..."` is the preferred top-level and step-local API document reference; `openapi = "..."` remains a compatibility alias. |
| Infer source types from package paths | `[+]` | `openapi/` maps to `openapi`, `google-discovery/` and legacy `discovery/` map to `google-discovery`, and `aws-smithy/` maps to `aws-smithy`. |
| Package all first-class source directories | `[+]` | Package inventory, digest construction, trusted-runner staging, and review handoff include `openapi/`, `google-discovery/`, `aws-smithy/`, and legacy `discovery/`. |
| Keep runtime semantics out of OpenUdon | `[+]` | OpenUdon still hands typed UWS plus staged source files to a trusted executor; provider auth, protocol serialization, and credentials remain executor-owned. |

## Verification

- `go test ./...`
- `go run ./cmd/openudon check-doc-memory`
- `go run ./cmd/openudon check-apitools-boundary`
- `git diff --check`
