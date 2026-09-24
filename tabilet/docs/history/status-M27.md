# Retired milestone M27 - M27 — UWS 1.2 First-Class API Source Types

**Milestone.** M27
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M27.md
**Source status SHA-256.** 209bf5da7e0357a9d6eb648de1e35e66645deec3c7b622392a3e8366ad3c3f44
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
~~~~~~~~~~~~~~~~~~~~
