# Ramen To OpenUdon Intent Migration

## Purpose

This note records the migration idea discussed from the `w8m` session before switching work to
`../openudon`.

This note is historical and is superseded by `openudon.md` for the current OpenUdon/Ramen boundary.
The current boundary is stricter than this original migration sketch: OpenUdon owns IaC intent and
`.tf` generation, while Ramen only reuses generic/domain-neutral intent authoring mechanics for its
own workflow and iCoT implementation.

The goal is to reduce the learning curve for `openudon` adoption. Although users can author
`.tf` / OpenTofu-style files directly, many users will need help turning a high-level infrastructure
goal plus OpenAPI documents into usable `openudon` artifacts. Ramen already has useful public-ish
intent, iCoT, OpenAPI discovery, prompt, and refinement ideas. The corrected proposal is to keep
OpenUdon's IaC-specific intent model and `.tf` generator in OpenUdon, while sharing only generic
intent/session primitives that Ramen can use for its private workflow domain.

## Current Relationship

- Ramen is a private workflow artifact generation, validation, eval, and handoff layer around
  Symphony, UWS, and `udon`.
- `openudon` is the public API-backed IaC authoring, planning, public contract, and handoff-bundle
  layer.
- `w8m` is the private trusted IaC executor that consumes approved `openudon` artifacts and runs
  through private `udon`.
- Ramen should remain orchestration and workflow-project glue. It should not own public IaC intent
  contracts and should not need to understand how OpenUdon IaC works.

Recent relevant commits at the time of this note:

- `../openudon`: `5eacffb Document public executor handoff`
- `../w8m`: `3671fb7 Document executor packaging boundary`
- `../ramen`: `ac3e7bc Require openapisearch sibling`

## Proposed OpenUdon Intent Core

Add an `openudon` intent layer that can help users produce public IaC artifacts from a higher-level
goal:

```text
user goal / project brief
  -> openudon intent
  -> suggested .tf / graph / profile skeleton / plan inputs
  -> public openudon handoff bundle
  -> private w8m execution later
```

The intent core should be openudon-native. It should know API-backed IaC concepts, public profiles,
OpenAPI operation mapping, state/plan/handoff artifacts, and review constraints. It should not know
Ramen, Symphony, private `w8m` request schemas, private credential policy, private profile packs, or
`udon` execution internals.

Possible package shape in `../openudon`:

```text
intent/v1alpha1        # public intent document types and validation
internal/intentgen     # optional AI-assisted generation/refinement
internal/openapidisco  # public-safe OpenAPI discovery/ranking
internal/handoff       # public bundle assembly
cmd/openudon           # future CLI: intent, plan, export-bundle
```

## Current Boundary

Ramen is not an IaC owner. It should not import OpenUdon's concrete IaC packages such as
`intent/v1alpha1`, `intent/tfgen`, graph, profile, plan, or handoff packages for its private
workflow/iCoT path.

Ramen may import generic/domain-neutral OpenUdon `intent` package-family abstractions to avoid
duplicating shared authoring mechanics: sessions, slots, transcripts, artifacts, diagnostics,
symbolic binding references, and provider-neutral chat interfaces.

OpenUdon owns the concrete IaC path: project brief or future OpenUdon iCoT/chat draft to reviewed
IaC intent, deterministic `.tf`, profiles, plans, and public handoff artifacts.

## What To Extract From Ramen

Good candidates for extraction or adaptation:

- Project brief parsing patterns and policy-section structure.
- Guided iCoT slot filling.
- OpenAPI discovery, candidate ranking, and operation metadata summarization.
- Operation/parameter inference from project text and OpenAPI metadata.
- Credential binding name handling, while keeping secret values out of generated artifacts.
- No-secret artifact checks.
- Structured LLM client interfaces and prompt/refinement patterns.
- Review evidence patterns that are generic to generated artifacts.
- Eval fixture style, if rewritten as openudon-specific IaC fixtures.

The extraction should be selective. Do not move Ramen internals wholesale. Reshape the reusable parts
around `openudon` concepts: IaC resources, data sources, public profiles, desired attributes,
identity fields, state snapshots, apply-state plans, refresh plans, and public handoff bundles.

## What Must Stay In Ramen Or Udon

Keep these out of `openudon` intent core:

- Symphony handoff and approval workflow.
- `ramen run`, trusted runner, approval templates, and readiness/private checkout reports.
- Generic `workflow.hcl` generation.
- UWS structural workflow semantics.
- `udon` runtime/profile validation and execution behavior.
- Ramen release/eval harness unless adapted into openudon-specific public fixtures.
- Command, SSH, function-adapter, or generic workflow runtime policy.

Boundary rule:

```text
If it helps produce public API-backed IaC artifacts, it can move toward openudon.
If it manages Symphony, generic workflows, UWS execution, trusted running, or private policy, it stays
in Ramen, udon, w8m, or the appropriate sibling.
```

## Suggested Migration Slices

1. In `openudon`, define `intent/v1alpha1` with a small public IaC intent model and fixture tests.
2. Port only public-safe OpenAPI discovery/ranking helpers needed for IaC resource/profile
   suggestions.
3. Add deterministic generation from intent to `.tf`/graph/profile skeletons before adding AI
   assistance.
4. Add optional AI-assisted `internal/intentgen` using a narrow interface and no provider-specific
   secret handling.
5. Add `internal/handoff` bundle assembly and a future `cmd/openudon export-bundle`.
6. Keep Ramen integration limited to generic OpenUdon `intent` package-family reuse for shared
   authoring mechanics.
7. Keep `w8m` invocation outside both public `openudon` and extracted intent code; private
   operators supply request metadata, credential policy, profile pack, ledger, output directory, and
   binary/container execution.
