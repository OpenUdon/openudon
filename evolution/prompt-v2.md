# Evolution Prompt v2

OpenUdon's direction changes from a private Symphony integration layer to the public UWS workflow
authoring, review, package, and executor-handoff tool.

## Target Boundary

1. Human/project input starts in `project.md`.
2. OpenUdon's structured authoring contract is `workflows/intent.hcl`.
3. `workflows/workflow.hcl` and `workflows/workflow.uws.yaml` are equivalent full UWS Document
   serializations, not separate public/private semantic layers.
4. Referenced API descriptions live in `openapi/*.yaml`.
5. Review, quality, approval, package digest, credential policy, and handoff evidence live in
   `expected/*.json` and `expected/*.md`.
6. `openudon run` validates approval and package digest, then hands the validated UWS/OpenAPI package
   plus non-secret executor config to a private trusted executor.

## Repo Ownership

- `../uws` owns public UWS parsing, validation, model, and portable workflow semantics.
- `../apitools` is narrowed to OpenAPI document search, discovery, import, download, local scanning,
  operation indexing, summaries, auth/security summaries, and ranking.
- `../openudon` owns iCoT, prompt transcript/replay, JSON completion fallback, artifact sets, review
  evidence, approval states, package digests, credential-value scanning, symbolic binding
  contracts, handoff validation, and trusted executor handoff.
- `../udon` remains a private executor boundary, ideally invoked through CLI/Docker-compatible
  package handoff rather than broad Go imports.
- `../symphony` is optional orchestration for workspaces, reviewer routing, identity, state
  transitions, and audit persistence.

## Migration Request

Move OpenUdon-owned lifecycle helpers out of `../apitools` and into `../openudon`, while keeping OpenAPI
discovery/search/import/indexing and prompt-safe operation summaries in `../apitools`. Keep the
existing `apitools.review-handoff.v1` manifest wire version during migration to avoid artifact
churn.
