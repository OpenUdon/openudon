# Synthesize And Build

The synthesis path starts from a reviewed project brief and produces the package artifacts that
reviewers inspect before any trusted-runner handoff.

## Generate A Package

```bash
export OPENUDON_LLM_PROVIDER=copilot-api
export OPENUDON_LLM_MODEL=gpt-5.4-mini

go run ./cmd/openudon synthesize \
  --example ./examples/<name> \
  --provider "$OPENUDON_LLM_PROVIDER" \
  --model "$OPENUDON_LLM_MODEL" \
  --max-attempts 5
```

`synthesize` reads `project.md`, API/event source files under `openapi/`, `google-discovery/`,
`aws-smithy/`, `asyncapi/`, `graphql/`, `openrpc/`, `grpc-protobuf/`, and `odata/` plus optional existing intent. It
writes:

```text
workflows/intent.hcl
workflows/workflow.hcl
workflows/workflow.uws.yaml
expected/plan.json
expected/plan.md
expected/discovery.json
expected/refinement.json
expected/refinement.md
expected/review.md
expected/review-handoff.json
expected/quality.json
expected/quality.md
```

## Provider Catalog Inputs

Before adding a hand-written OpenAPI slice or searching public catalogs, inspect the first-class
provider catalog:

```bash
go run ./cmd/openudon catalog inspect stripe
go run ./cmd/openudon catalog advisory --example ./examples/<name>
```

If the provider has a directly importable API source reference, import or materialize it into the package-local
source-aligned directory before running `synthesize`:

```bash
go run ./cmd/openudon catalog import-openapi --provider stripe --example ./examples/<name>
```

Catalog OpenAPI/Swagger, Google Discovery, AWS Smithy JSON, AsyncAPI, GraphQL, OpenRPC,
gRPC/protobuf, and OData artifacts are first-class API/event source inputs for generated UWS
artifacts. Stone, Postman Collection, RAML, API
Blueprint, human-docs, and unknown protocols remain advisory or OpenAPI-lowering/review inputs.

OpenUdon validates and packages AsyncAPI source-bound UWS 1.3 workflows and GraphQL/OpenRPC/gRPC/OData
source-bound UWS 1.4 workflows. It does not implement protocol serialization, broker subscription,
message delivery, RPC transport, endpoint selection, or credential resolution; those remain
trusted-runtime responsibilities.

## Narrower Stages

Use `build` after editing `workflows/intent.hcl`:

```bash
go run ./cmd/openudon build --example ./examples/<name> --max-attempts 5
```

Use `promote` after editing `workflows/workflow.hcl`:

```bash
go run ./cmd/openudon promote --example ./examples/<name>
```

Use `assess` to rerun deterministic quality gates only:

```bash
go run ./cmd/openudon assess --example ./examples/<name>
```

## Review Next

Generated artifacts are evidence, not approval. Reviewers should inspect `expected/quality.md`,
`expected/review.md`, `expected/plan.md`, and `expected/review-handoff.json`, then generate
approval JSON only when the package is acceptable for the intended tier.
