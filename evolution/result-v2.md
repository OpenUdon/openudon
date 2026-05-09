# Evolution Result v2

The public OpenUdon boundary is accepted.

## Accepted Direction

- OpenUdon is the public UWS authoring, review, package, and executor-handoff tool.
- Symphony is optional orchestration, not the product boundary.
- `workflow.hcl` and `workflow.uws.yaml` are equivalent full UWS Document serializations.
- `apitools` should narrow back to OpenAPI document tooling.
- OpenUdon owns product-level lifecycle APIs even when they look reusable: iCoT, prompt transcripts,
  JSON completion fallback, artifact sets, review evidence, approval policy, package digest,
  credential scanning, symbolic binding contracts, and handoff validation.
- udon remains private execution infrastructure behind a trusted handoff path.

## Sequencing

1. Move low-risk review/package helpers into OpenUdon and keep the handoff wire shape stable.
2. Move iCoT lifecycle and JSON completion fallback into OpenUdon while keeping OpenAPI operation
   summaries and ranking in apitools.
3. Slim apitools to OpenAPI discovery/search/import/indexing plus compatibility shims only where
   needed.
4. Define the udon executor handoff as UWS Document + OpenAPI files + non-secret run config +
   runtime credential resolver, with Docker as the preferred packaging target.

## Current Result

OpenUdon now has local `internal/authoring` helpers for progressive iCoT lifecycle, draft/transcript
persistence, prompt replay types, JSON completion fallback, artifact/review metadata, symbolic
binding contracts, credential scanning, review handoff validation, and handoff package digests.
The manifest version remains `apitools.review-handoff.v1` for compatibility.
