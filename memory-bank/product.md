# Product

## Memory Bank Index

- This file owns product purpose, audience, workflows, scope, and non-goals.
- Use [architecture.md](architecture.md) for system boundaries, data flow, and planned structure.
- Use [tech-stack.md](tech-stack.md) for implementation technologies and dependency constraints.
- Use [milestone.md](milestone.md) for milestones, work sequencing, and acceptance criteria.
- Use [status.md](status.md) for current completion state.

Ramen is the public-facing UWS workflow authoring, review, package, and executor-handoff tool. It
can be used directly by operators or under optional Symphony-managed orchestration, and it hands
approved packages to a trusted executor boundary such as the private `udon` runtime.

Ramen turns reviewed project briefs into deterministic workflow artifacts: `project.md`,
`workflows/intent.hcl`, `workflows/workflow.hcl`, exported UWS, expected plans, quality reports,
review evidence, refinement reports, package digests, credential policy, and machine-readable
review handoff manifests. It does not own public workflow semantics or generic execution. Those
remain in `../uws` and executor implementations such as `../udon`.

## Product Goal

Make UWS workflow projects authorable, reviewable, packageable, and executable only through a
validated trusted handoff path, with clear evidence for every generated artifact and side-effect
boundary.

## Primary Users

- Operators authoring or reviewing UWS/OpenAPI workflow projects.
- Optional Symphony-managed agents generating artifacts inside isolated workspaces.
- Reviewers checking side effects, credential bindings, generated plans, quality reports, and
  approval states before execution.
- Runtime operators using `ramen run` to validate approval and package digests before invoking a
  trusted executor.
- Ramen maintainers extending prompts, iCoT, eval fixtures, quality gates, and cross-repo glue.

## Core Workflows

1. Author or refine an example brief under `examples/<name>/project.md`.
2. Use iCoT to guide authoring and produce `project.md` plus `workflows/intent.hcl`.
3. Synthesize, build, promote, and assess deterministic artifacts from reviewed inputs.
4. Validate OpenAPI availability, intent shape, workflow compilation, UWS export, expected-plan
   matching, review evidence, credential policy, and secret scanning.
5. Run eval fixtures to compare prompt/model/pipeline behavior across curated briefs.
6. Generate local readiness evidence for private sibling checkout state and deterministic gates.
7. Produce approval JSON from the current handoff package digest.
8. Use `ramen run` to validate handoff, stored and current quality, approval state, package digest,
   tier compatibility, and trusted executor invocation.

## Core Concepts

- **Project brief** is the human policy and workflow source in `project.md`.
- **Intent** is Ramen's structured `workflows/intent.hcl` contract for workflow metadata, inputs,
  steps, outputs, data-flow hints, credentials, runtime approvals, side-effect scope, timeouts, and
  idempotency metadata.
- **Workflow HCL** is one full UWS Document serialization generated from intent.
- **UWS YAML** is an equivalent full UWS Document serialization of the same workflow. Ramen treats
  `workflow.hcl` and `workflow.uws.yaml` as public UWS documents, not separate semantic layers.
- **Expected plan** records inferred steps, runtimes, OpenAPI operations, dependencies, request
  inputs, bindings, credentials, structural results, side effects, and action policies.
- **Quality report** is the deterministic release gate for current generated artifacts.
- **Review evidence** is the human-readable side-effect, risk, credential, and trusted-runner
  package summary.
- **Review handoff** is an `apitools.review-handoff.v1` manifest with Ramen package inputs,
  approval states, owner split, execution policy, credential binding names, and trusted runner
  metadata. The wire version remains stable during the migration.
- **Trusted runner** is the local `ramen run` gate that invokes a private executor path only after
  approval and package validation pass.

## Scope

- Ramen-owned project templates, prompts, guided iCoT authoring, examples, eval fixtures, and review
  policy.
- Deterministic artifact generation from project briefs and intent into workflow HCL, UWS, plan,
  quality, refinement, review, and handoff artifacts.
- Quality gates and local validation wrappers for Ramen package correctness.
- Optional Symphony workflow prompt/config policy for Ramen-managed work items.
- Local trusted execution wrapper, approval template generation, package digest checks, and tier
  enforcement.
- Cross-repo compatibility evidence for UWS semantics, udon lowering/runtime behavior, provider
  drift, release gates, and private checkout readiness.
- Ramen-owned iCoT, review evidence, approval, package digest, credential policy, and trusted
  executor handoff helpers.
- OpenAPI discovery/search/import/indexing reuse through `github.com/OpenUdon/apitools`.

## Non-Goals

- Public UWS schema or semantics; those belong in `../uws`.
- Generic OpenAPI/UWS compilation, lowering, or runtime execution behavior; those belong in
  executor implementations such as `../udon`.
- Symphony service implementation, reviewer identity storage, managed state transitions, or audit
  persistence; those belong in `../symphony`.
- Concrete OpenUdon IaC intent models, `.tf` generation, graph/profile/planning/state/drift
  behavior, or `w8m` executor contracts; those belong in `../openudon` and private executors.
- Secret storage, credential resolution, account selection, endpoint selection, production
  execution policy, or provider SDK ownership.
- Direct production side effects from synthesis, build, promote, assess, iCoT, or eval commands.
