# Alpha Prompt v1

## Project Context

Design the private Ramen integration layer for Symphony-managed UWS/OpenAPI workflow projects
executed by the private `udon` runtime.

Ramen should make project ideas reviewable and executable only through deterministic artifact
generation, quality evidence, approval handoff, and trusted runtime enforcement.

## Core Requirements

1. Keep Ramen thin: public workflow semantics belong in `../uws`, generic UWS/OpenAPI compilation
   and execution belong in `../udon`, and work orchestration belongs in `../symphony`.
2. Support guided authoring from project idea to `project.md` and `workflows/intent.hcl`.
3. Generate `workflow.hcl`, exported UWS, expected plan, discovery, refinement, review, quality,
   and handoff artifacts from reviewed inputs.
4. Validate generated artifacts deterministically before any execution path.
5. Maintain an eval corpus and release evidence for prompt/model/pipeline changes.
6. Use `github.com/OpenUdon/apitools` for shared authoring mechanics and public review handoff schema
   where those concepts stay domain-neutral.
7. Keep side-effectful execution behind approval JSON, package digest validation, tier checks,
   current quality validation, and trusted udon invocation.
8. Keep secrets, credential values, private endpoints, and production side effects out of prompts,
   examples, committed artifacts, and agent sessions.

## Success Criteria

- A trusted operator can author a project, synthesize artifacts, inspect quality/review evidence,
  create approval JSON, and run a dry trusted-runner gate from documented commands.
- Generated artifacts are deterministic enough for review and release comparison.
- Quality failures identify concrete repair targets.
- Ramen consumes sibling public/generic contracts without absorbing their ownership.
- Symphony can route review from Ramen evidence, but Ramen does not need to fork Symphony.

## Product Direction

Build Ramen as the private workflow-integration and review-handoff layer. Keep public semantics,
runtime execution, IaC behavior, and managed orchestration in the sibling projects that own them.
