# Architecture

## Memory Bank Index

- This file owns system boundaries, data flow, planned structure, and security boundaries.
- Use [product.md](product.md) for product scope and non-goals.
- Use [tech-stack.md](tech-stack.md) for implementation technologies.
- Use [milestone.md](milestone.md) for milestones and acceptance criteria.
- Use [status.md](status.md) for current completion state.

Ramen is an integration layer above Symphony, UWS, udon, and shared `apitools`. It owns the
Ramen-specific workflow package lifecycle from brief authoring through review handoff and trusted
local execution. It composes sibling projects but should stay thin.

## Current State

Ramen has a Go module, a thin `cmd/ramen` CLI, a guided `cmd/icot` authoring CLI, deterministic
synthesis/build/promote/assess commands, an eval harness, local readiness reporting, and a trusted
runner wrapper. It emits reviewed package artifacts under each example directory and validates those
artifacts before any approved udon execution path.

Generated packages now include project briefs, structured intent, workflow HCL, UWS YAML, expected
plans, OpenAPI discovery reports, refinement reports, review notes, quality reports, and
`expected/symphony-handoff.json` manifests built on the public `apitools` review handoff schema.

## System Boundary

- `../uws` owns public workflow semantics, UWS versions, schema, and Go model.
- `../udon` owns generic UWS/OpenAPI compilation, lowering, execution, runtime profiles, and
  runtime-plan behavior.
- `../apitools` owns domain-neutral authoring helpers, OpenAPI discovery/import helpers, review-only
  leaf adapters, public review state machine, and runtime-neutral handoff schema.
- `../openudon` owns public IaC authoring/planning, concrete IaC intent, `.tf` generation, graph,
  profile, state, drift, and `w8m`-facing public artifacts.
- `../symphony` owns work orchestration, isolated workspaces, reviewer routing, identity, managed
  state transitions, and audit persistence.
- Ramen owns only private workflow integration: project policy, examples, prompts, iCoT workflow
  authoring, Symphony agent instructions, deterministic Ramen validation, review evidence, approval
  templates, and local trusted-runner enforcement.

## Cross-Repo Contract Summary

Ramen must not teach prompts to emit workflow semantics that lack a public UWS contract. UWS 1.1
defines portable timeout fields and workflow-level idempotency metadata; Ramen may preserve those
only when project policy or intent explicitly requests them. Switches, loops, structural results,
failure branches, retries, and runtime profiles are either public UWS constructs or extension-owned
profiles, but Ramen still needs udon compatibility proof and project policy before making them
generation defaults.

Current generation policy:

| Capability | Ramen generation policy |
| --- | --- |
| Switch branches | Allowed; Ramen has prompt, plan, review, and quality support. |
| Loops | Allowed; Ramen proves loop lowering, plan/review evidence, UWS export, and quality coverage. |
| Structural results | Allowed for generated structural step outputs and validated against expected plans. |
| Failure branches | Allowed only when brief or intent explicitly asks for failure routing. |
| Retries | Allowed only when explicitly requested; side-effectful retries need retry/idempotency policy. |
| Timeouts | Allowed only when explicit `ramen-policy` or intent metadata requests them. |
| Idempotency | Allowed for explicit workflow-level UWS 1.1 metadata; Ramen does not inject API keys. |
| Runtime profiles | Allowed only for existing validated udon profile shapes and project/environment policy. |

Closed cross-repo dependencies remain regression responsibilities:

- Structured output and UWS artifact preservation regressions are watched in Ramen and udon tests.
- Rich OpenAPI behavior is covered by Ramen eval fixtures first; reusable gaps move to `../udon`
  only after concrete failures.
- Symphony approval handoff is closed in Ramen through emitted artifacts and the trusted wrapper;
  managed reviewer routing remains optional external Symphony work.
- Provider drift is reported in eval JSON/Markdown and release notes.
- Private checkout and secret automation remain local/manual through readiness reports until the
  private dependency layout and redaction controls stabilize.
- Runtime/profile coverage stays as Ramen policy/eval evidence unless a reusable UWS/udon semantic
  gap is proven.

## System Flow

1. A trusted user or Symphony-managed issue starts from a natural-language project brief.
2. iCoT may guide the user through goal, OpenAPI, operation, inputs, outputs, credential bindings,
   side-effect scope, safety, and fallback questions.
3. Ramen saves `project.md` and `workflows/intent.hcl`.
4. `ramen synthesize` discovers/imports local OpenAPI inputs, generates intent when needed, builds
   workflow HCL through udon, exports UWS, and writes plan, refinement, review, handoff, and quality
   artifacts.
5. `ramen build`, `ramen promote`, and `ramen assess` rerun narrower stages after edits.
6. Quality gates validate project policy, OpenAPI availability, intent validity, data flow, workflow
   compilation, expected-plan matching, UWS profile/schema checks, review evidence, credential
   policy, side-effect policy, handoff contract, and secret scanning.
7. Reviewers inspect the minimum review package and, when appropriate, create approval JSON for
   sandbox or production tier.
8. `ramen run` revalidates the package, current quality, approval JSON, digest, tier/state rules,
   credential-value policy, and direct-production policy before invoking `scripts/run-udon.sh` by
   argv.

## Artifact Flow

- Inputs: `project.md`, local OpenAPI files under `openapi/`, optional existing
  `workflows/intent.hcl`, provider credentials in environment variables, and sibling schemas.
- Authoring outputs: `workflows/intent.hcl` and regenerated `project.md` from iCoT reconcile.
- Generated outputs: `workflows/workflow.hcl`, `workflows/workflow.uws.yaml`,
  `expected/plan.json`, `expected/plan.md`, `expected/discovery.json`,
  `expected/refinement.json`, `expected/refinement.md`, `expected/review.md`,
  `expected/symphony-handoff.json`, `expected/quality.json`, and `expected/quality.md`.
- Eval outputs: ignored JSON/Markdown reports and optional archived workspaces under `eval/runs/`
  and `eval/artifacts/`.
- Readiness outputs: ignored local readiness JSON under `eval/readiness/`.
- Trusted-runner outputs: local approval JSON and workdir artifacts under ignored operator paths.

## iCoT Architecture

iCoT is Ramen's interactive thinking layer, not a synthesis or execution engine. It preserves the
same source artifacts as normal Ramen examples: `project.md` for human policy and
`workflows/intent.hcl` for the structured workflow contract.

The LLM-assisted path asks one broad goal question, drafts intent from local context and OpenAPI
metadata, runs deterministic readiness checks, and asks only the highest-priority blocking
follow-up. The offline path uses the fixed manual loop. iCoT autosaves local session state under
ignored `.icot/session.yaml`, writes ignored transcripts when enabled, and treats final artifact
writes as an atomic small transaction.

The conversation engine is not Ramen-owned. `apitools.RunProgressiveICOT[S, D, A]` plus
`ProgressiveLoopHooks[S, D, A]` are the bound-runtime layer (mirroring the
`uws1.Document.Runtime` pattern): apitools owns the structural loop — opening, optional
disambiguation, draft+question iterations, final confirmation, transcript persistence — and
each engine binds its own domain-shaped hooks. Ramen's `internal/icot/elicitor/` files
(`extractor`, `classification`, `progressive`, `loop`, `session`, `api`) are the rollout
binding. `apitools/openapidisco` and `apitools/icot` (`LoadDraft[T]`/`SaveDraft[T]`)
are the cross-engine helpers; OpenUdon will plug an IaC-shaped binding into the same
generic loop.

## Symphony And Trusted Execution

Ramen keeps `../symphony` untouched. The Ramen-owned handoff package gives Symphony or any reviewer
enough evidence to route review, but Ramen does not implement managed reviewer identity or audit
history.

The local trusted runner is intentionally separate from synthesis. It validates
`expected/symphony-handoff.json`, `expected/quality.json`, current in-memory quality, approval JSON,
canonical package digest, and tier compatibility. It rejects credential values in artifacts and
direct production execution, then invokes udon only through `scripts/run-udon.sh`.

Required handoff inputs are `project.md`, `workflows/intent.hcl`, `workflows/workflow.hcl`,
`workflows/workflow.uws.yaml`, `expected/plan.json`, `expected/quality.json`,
`expected/refinement.json`, `expected/review.md`, and `expected/symphony-handoff.json`. Approval
states are `generated`, `validated`, `review_required`, `approved_for_sandbox`,
`approved_for_production`, and `rejected`.

Automation tiers:

| Tier | Gate | Location |
| --- | --- | --- |
| Local deterministic | `go test ./...`, `go vet ./...`, `make check`, `git diff --check` | Trusted workstation with private siblings. |
| Local readiness report | `ramen readiness --run-gates --out eval/readiness/local.json` | Trusted workstation with private siblings. |
| Local/manual real LLM | `ramen eval --release-gate` or `make release-eval` | Trusted workstation with provider env vars. |
| Future protected automation | New design required | Protected runner only after checkout and redaction controls stabilize. |

## Planned File And Folder Structure

- `cmd/ramen/`: thin CLI for check, synthesize, build, promote, assess, eval, readiness, approval
  template, and trusted run commands.
- `cmd/icot/`: guided Ramen authoring CLI.
- `internal/synthesize/`: artifact generation, expected plans, quality gates, review evidence,
  refinement loop, and Symphony handoff manifests.
- `internal/icot/`: interactive authoring session, reconcile, lint, replay, extraction, and prompt
  handling.
- `internal/eval/`: fixture eval, reference comparison, run comparison, reporting, release gates,
  and provider drift watch.
- `internal/trustedrunner/`: approval schema, package digest, handoff validation, tier checks, and
  udon invocation wrapper.
- `internal/readiness/`: local private checkout readiness reports and deterministic gate execution.
- `internal/workflowintent/`: Ramen compatibility adapter over shared generic authoring concepts.
- `examples/`: committed examples and eval corpus.
- `templates/`: project brief starter templates.
- `memory-bank/`: living project memory.
- `evolution/`: versioned prompt/result snapshots for milestone-level direction changes.

## Security Boundary

Generated UWS, OpenAPI, HCL, review, and approval artifacts are untrusted until validated. Ramen
must not put secrets in prompts, examples, eval fixtures, committed artifacts, or logs. Credential
bindings are symbolic names only until a trusted runtime resolves them. Production side effects are
never allowed from agent sessions, synthesis, build, promote, assess, iCoT, or eval.
