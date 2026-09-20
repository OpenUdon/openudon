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
