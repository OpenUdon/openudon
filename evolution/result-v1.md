# Alpha Result v1

## Product Direction

Ramen is the private workflow package lifecycle layer for Symphony-managed UWS/OpenAPI projects. It
turns project briefs into reviewed artifacts and allows udon execution only through trusted local
approval gates.

## Architecture Pattern

- `project.md` is the human-readable project policy and brief.
- `workflows/intent.hcl` is the Ramen-owned structured workflow contract.
- udon generates and validates executable workflow HCL and UWS artifacts.
- Ramen records expected plans, quality reports, refinement attempts, review evidence, and
  machine-readable handoff manifests.
- `apitools` provides domain-neutral review state/handoff contracts and shared authoring mechanics.
- Symphony remains an external work orchestration service.
- `ramen run` is the local trusted gate before udon invocation.

## MVP Boundary

1. Guided iCoT authoring of project and intent artifacts.
2. Deterministic synthesis, build, promote, and assess commands.
3. Quality gates for policy, data flow, OpenAPI, workflow, UWS, review, credentials, side effects,
   handoff, and secret scanning.
4. Eval and readiness evidence for local/manual release confidence.
5. Approval template generation and trusted-runner validation.

## Deferred

- Public UWS semantics.
- Generic runtime execution or compiler features.
- Symphony reviewer identity, audit persistence, and managed routing implementation.
- OpenUdon concrete IaC models or `.tf` generation.
- Credential resolution, production endpoint selection, and direct production execution from agent
  sessions.
