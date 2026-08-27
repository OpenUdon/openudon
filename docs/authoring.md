# Authoring

OpenUdon has two supported authoring paths. Both produce the same reviewable package shape: a
human-readable `project.md`, a structured `workflows/intent.hcl`, public UWS artifacts, expected
plans, review evidence, quality reports, and a handoff manifest.

## Path 1: Guided iCoT

Use iCoT when you want an operator-guided session that starts from a goal and writes the initial
brief plus intent.

```bash
go run ./cmd/icot --example ./examples/<name>
```

iCoT writes an approved `project.md` plus either `workflows/intent.hcl` or an explicitly incomplete
`workflows/intent.draft.hcl`; it does not execute workflows. It can run with optional LLM assistance,
without LLM extraction, from an existing example, or from an `openudon.icot-session.v2` YAML/JSON
session. The adaptive interview shows every dependency-ready decision in a frontier round and keeps
later workflows as unnumbered candidates. Use `--prompt-mode full|normal|fast` to choose between full
questioning, visible safe defaults, or silent safe defaults; final proposal approval is always
explicit unless `--yes` is supplied.

After iCoT saves artifacts, continue with:

```bash
go run ./cmd/openudon build --example ./examples/<name>
go run ./cmd/openudon assess --example ./examples/<name>
```

Use [iCoT](icot.md) for command details and [Project Briefs](project-authoring.md) for the
sections a good `project.md` should contain.

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
descriptions when the trusted executor supports them. AsyncAPI source-bound workflows emit UWS 1.3;
GraphQL, OpenRPC, gRPC/protobuf, and OData source-bound workflows emit UWS 1.4. OpenUdon validates
and packages those source-bound workflows, but protocol execution remains trusted-runtime-owned.
`build` regenerates from existing intent.
`assess` reruns deterministic quality checks without synthesizing new intent.

Operators may add a `content_trust` block to `workflows/intent.hcl` after the
workflow and source choices are reviewed. OpenUdon maps those declarations to
the generated UWS 1.9.1 document. This block is deliberately operator-authored,
not an LLM-generation field. Projects without it retain their prior UWS
version and package shape; browser 1.7 by itself continues to use UWS 1.9.0.
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
