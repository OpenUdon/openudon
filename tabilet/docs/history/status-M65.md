# Retired milestone M65 - Status M65 - iCoT M29 Follow-Up Closure

**Milestone.** M65
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M65.md
**Source status SHA-256.** ba21f6c43e9bac492e9de9ecea59573db75de85496c5acf59bcdc6367f68a06d
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
# Status M65 - iCoT M29 Follow-Up Closure

State of the three OpenUdon iCoT follow-up tasks left from M29.

## Goal

Turn the deferred M29 response metadata, deterministic API prework, and focused
replay-quality follow-ups into concrete bounded behavior without making iCoT an
unbounded autonomous planner.

## Tasks

| Item | State | Notes |
|---|---|---|
| Response-field metadata expansion | `[+]` | Review-repair response-field extraction now resolves local OpenAPI `$ref` schemas, nested object fields, arrays, and Swagger `schema` responses before binding local `fnct` inputs. Google Discovery and AWS Smithy response fields still require source-family-specific evidence before broader claims. |
| Deterministic API prework generalization | `[+]` | Review repair can add one deterministic read-only API prework step when exactly one local GET/HEAD operation can produce the missing request field and has no required inputs; ambiguous, credentialed, or input-requiring prework still rejects. |
| Replay quality gate polish | `[+]` | Added `make icot-replay-repair-check` as a focused replay-repair gate over known `cmd/icot replay-eval --prompt-mode fast --review-repair` fixtures, keeping it explicit rather than hidden inside every release check. |

## Acceptance Criteria

- iCoT only adds API prework from listed local metadata.
- Added prework is read-only and does not invent request inputs,
  credentials, providers, or operations.
- Response-field bindings remain schema-backed and bounded.
- Replay repair has a focused named local gate.

## Verification

- `go test ./internal/icot/elicitor -run 'TestApplyDraftReviewRemediations'`
- `make -n icot-replay-repair-check`
- `go test ./internal/icot ./internal/icot/elicitor ./cmd/icot`
- `go test ./...`
- `go vet ./...`
- `go run ./cmd/openudon check-doc-memory`
- `make check`
- `go run ./cmd/openudon check-apitools-boundary`
- `git diff --check`
- `git -C ../tofu diff --check -- openudon`
~~~~~~~~~~~~~~~~~~~~
