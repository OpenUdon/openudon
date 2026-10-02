# Authoring

OpenUdon has two supported authoring paths. Both produce the same reviewable package shape: a
human-readable `project.md`, a structured `workflows/intent.hcl`, public UWS artifacts, expected
plans, review evidence, quality reports, and a handoff manifest.

## Path 1: Kinet or explicit local seed

Use Kinet for interviews and interactive workflow authoring. OpenUdon supplies
native step, source, supervised capture and package commands; it has no UI.
For deterministic local drafting from reviewed inputs:

```bash
go run ./cmd/openudon authoring draft \
  --from-example ./examples/eval/runtime-only-render \
  --example .openudon-run/authoring-example --prompt-mode fast --no-llm --yes
go run ./cmd/openudon build --example .openudon-run/authoring-example
go run ./cmd/openudon assess --example .openudon-run/authoring-example
```

Print is read-only, complete seed publication requires explicit yes, and partial
input returns a structured frontier without terminal questions or writes. See
[current authoring migration](authoring-retirement.md),
[step contracts](step-authoring-contract-v1.md) and
[Project Briefs](project-authoring.md). Existing session/report artifact formats
remain readable; optional expert model evaluation retains its separate authority.

## Path 2: Brief And Synthesis

Use synthesis when you already have a project brief or are updating a fixture.

```bash
go run ./cmd/openudon synthesize --example ./examples/<name>
go run ./cmd/openudon build --example ./examples/<name>
go run ./cmd/openudon assess --example ./examples/<name>
```

`synthesize` reads `project.md`, discovers or imports local API/event source metadata, creates or
updates intent, and writes the generated package artifacts. OpenAPI, Google Discovery, AWS Smithy
JSON, AsyncAPI, GraphQL, OpenRPC, gRPC/protobuf, and OData can be staged directly as UWS source
descriptions when the trusted executor supports them. New workflows declare UWS 1.12.0;
AsyncAPI binding was introduced in 1.3 and GraphQL, OpenRPC, gRPC/protobuf, and OData
binding in 1.4. OpenUdon validates
and packages those source-bound workflows, but protocol execution remains trusted-runtime-owned.
`build` regenerates from existing intent.
`assess` reruns deterministic quality checks without synthesizing new intent.

Operators may add a `content_trust` block to `workflows/intent.hcl` after the
workflow and source choices are reviewed. OpenUdon maps those declarations to
the generated UWS 1.12.0 document. This block is deliberately operator-authored,
not an LLM-generation field. Existing packages retain their declared UWS
versions and package shape.
See [intent.hcl](intent.md#content-trust) for the exact declaration syntax and
validation boundary.

Before searching public catalogs, inspect first-class provider metadata from `apitools`:

```bash
go run ./cmd/openudon catalog list
go run ./cmd/openudon catalog inspect github
go run ./cmd/openudon catalog advisory --example ./examples/<name>
```

When a provider has a directly importable OpenAPI reference, import it into the package-local
`openapi/` directory:

```bash
go run ./cmd/openudon catalog import-openapi --provider stripe --example ./examples/<name>
```

Discovery, Smithy, AsyncAPI, GraphQL, OpenRPC, gRPC/protobuf, and OData catalog entries can be
materialized as first-class API/event source inputs when a package needs them. Stone, Postman
Collection, RAML, API Blueprint, and human-docs catalog
entries remain advisory metadata until lowered or reviewed separately.

Use [Synthesize](synthesize.md), [intent.hcl](intent.md), and [Data Flow](data-flow.md) for the
artifact contracts.

## Agentic SaaS Authoring

For common SaaS workflows, use [Agentic SaaS Authoring](agentic-saas-authoring.md) as the contract.
The AI-assisted path can draft goals, operation choices, request mappings, credential binding names,
and unresolved assumptions. Use the [n8n Pattern Bridge](n8n-pattern-bridge.md) only as
service-priority and mapping evidence; the generated artifacts stay OpenUdon-native and continue
through deterministic validation, review, packaging, and trusted handoff.

Use [iCoT](icot.md) when the brief is not precise yet. Its guided loop starts from provider/catalog
metadata when available, asks for listed OpenAPI operation IDs, lets the LLM draft request field
sources from selected operation details, and asks the operator only for unresolved credential
bindings, mappings, response/output sources, or side-effect boundaries before saving source
artifacts.

## Safety Rules

- Put credential binding names in artifacts, never credential values.
- Keep side-effectful workflows in generated/review state until approval.
- Treat content-trust declarations as reviewed provenance metadata, not as a
  substitute for side-effect approval, credential policy, or runtime controls.
- Use sandbox proof-run language for examples that send email, write records, call commands, or
  otherwise produce effects.
- Use `openudon run --dry-run` to validate the handoff package without invoking the executor.
