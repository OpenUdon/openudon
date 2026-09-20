# Status M68 - iCoT Lifecycle Planning Alignment

State of the lifecycle hint and operation-detail planning alignment for iCoT
workflow drafts.

## Goal

Build lifecycle hints and sibling operation details from one deterministic
per-round plan while preserving the bounded prompt-detail contract and
source-correct behavior across API documents.

## Scope

- Seed lifecycle planning from selected workflow operations.
- Fall back to ranked operations when a first draft has no selected steps.
- Use the full prompt-safe catalog context only for lifecycle inference.
- Keep the detailed prompt context bounded to the final seed, sibling, and
  explicitly requested operation detail documents.
- Keep explicitly requested operation details as additions, not lifecycle
  seeds.
- Make no public CLI, package API, UWS wire-format, workflow-semantic, or
  trusted-execution change.

## Task Status

| Item | State | Notes |
|---|---:|---|
| Per-round lifecycle plan | `[+]` | Each draft round builds one plan from selected operations or ranked first-draft operations. |
| Single expansion pass | `[+]` | The plan constructs the full catalog prompt context once and calls shared Authoring lifecycle expansion once per seed. |
| Hint/detail alignment | `[+]` | Lifecycle hints and sibling detail refs are derived from the same expansion results, so first drafts no longer receive lifecycle-expanded details without corresponding hints. |
| Source-correct sibling refs | `[+]` | Sibling resolution uses the expanded candidate's source ID plus operation ID before producing a document-path detail ref; duplicate operation IDs cannot select a sibling from another document. |
| Bounded prompt context | `[+]` | The draft prompt continues to construct its `docs` and `prompt_context` from only the final detail documents. The full catalog remains a compact operation inventory and is not exposed as detailed metadata. |
| Requested detail posture | `[+]` | LLM-requested detail refs remain bounded additions and do not become lifecycle seeds. |
| Regression coverage | `[+]` | Tests cover first-draft hint/detail alignment, stable selected-operation behavior, source-correct duplicate operation IDs, and requested-detail seed isolation. |

## Scoped Commits

- Hermetic eval discovery follow-up to M34: `1cb5a54`
- Lifecycle plan alignment: `249e7b4`

## Verification

- `go test ./internal/icot/elicitor -run 'Lifecycle|RequestedDetail' -count=1`
- `go test ./internal/icot -run 'TestEvalReferenceSeedBuildMatrix|TestDiscoverReferencePolicyFixtures' -count=1`
- `go test ./...`
- `go vet ./...`
- `git diff --check`
- `git -C ../tofu diff --check -- openudon`
