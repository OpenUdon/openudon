# Architecture

## Memory Bank Index

- This file owns system boundaries, data flow, planned structure, and security boundaries.
- Use [product.md](product.md) for product scope and non-goals.
- Use [tech-stack.md](tech-stack.md) for implementation technologies.
- Use [milestone.md](milestone.md) for milestones and acceptance criteria.
- Use [status.md](status.md) for current completion state.

Ramen is the public UWS authoring, review, package, and executor-handoff layer above UWS, narrowed
OpenAPI tooling, optional Symphony orchestration, and private executor implementations. It owns the
workflow package lifecycle from brief authoring through review handoff and trusted local execution.

## Current State

Ramen has a Go module, a thin `cmd/ramen` CLI, a guided `cmd/icot` authoring CLI, deterministic
synthesis/build/promote/assess commands, an eval harness, local readiness reporting, and a trusted
runner wrapper. It emits reviewed package artifacts under each example directory and validates those
artifacts before any approved udon execution path.

Generated packages now include project briefs, structured intent, workflow HCL, UWS YAML, expected
plans, OpenAPI discovery reports, refinement reports, review notes, quality reports, and
`expected/symphony-handoff.json` manifests. The manifest wire version remains
`apitools.review-handoff.v1` during migration.

## System Boundary

- `../uws` owns public workflow semantics, UWS versions, schema, parsing, validation, and Go model.
- `../apitools` owns OpenAPI document search, discovery, import, download, local file scanning,
  operation indexing, operation summaries, auth/security summaries, and operation ranking.
- `../ramen` owns UWS authoring, iCoT/progressive loops, prompt transcripts/replay, artifact sets,
  review evidence, approval state policy, package digests, credential-value scanning, symbolic
  binding contracts, review handoff validation, package gates, and trusted executor handoff.
- `../udon` owns private UWS/OpenAPI compilation, lowering, execution, runtime profiles, and
  runtime-plan behavior. The target public Ramen boundary invokes udon through CLI/Docker-compatible
  executor handoff rather than broad Go library coupling.
- `../openw8m` owns public IaC authoring/planning, concrete IaC intent, `.tf` generation, graph,
  profile, state, drift, and `w8m`-facing public artifacts.
- `../symphony` optionally owns work orchestration, isolated workspaces, reviewer routing, identity,
  managed state transitions, and audit persistence.

Transitional debt:

- Ramen no longer imports `github.com/genelet/udon`; approved packages are handed to external executors through a portable run-config shim.
- Ramen imports `github.com/OpenUdon/apitools` only for narrowed OpenAPI tooling. Product lifecycle
  helpers such as iCoT, transcript, review, handoff, credential scan, package digest, and approval
  policy are Ramen-owned.

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
| Runtime profiles | Allowed only for existing validated UWS runtime supplement shapes and project/environment policy. |

The public UWS runtime supplement is a slim non-HTTP invocation selector for extension-owned
execution only. Public `x-uws-runtime` carries only `type`, `command`, `workingDir`, `function`,
`workflow`, and `arguments`. HTTP/OpenAPI operations must use core UWS OpenAPI binding fields plus
referenced OpenAPI documents; `type: http` in public `x-uws-runtime` is rejected rather than treated
as a runtime profile. Provider selection, credentials, client/security configuration, and
request/response schemas belong in runtime-private configuration or product-owned profiles, not
public `x-uws-*`. This is intentional because runtime auth/security shapes for `ssh`, `cmd`,
`fnct`, `fileio`, `sql`, `s3`, `smtp`, `dns`, `ldaps`, `scp`, `sftp`, and `llm` are
implementation-specific and usually appear as runtime-owned arguments or private runtime
configuration rather than a portable public config object. Udon's legacy private `x-udon-runtime`
remains a separate compatibility concern until udon migrates its public DTO/export surface.

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
4. `ramen synthesize` discovers/imports local OpenAPI inputs, generates intent when needed, builds public UWS HCL/YAML artifacts directly from intent, and writes plan, refinement, review, handoff, and quality
   artifacts.
5. `ramen build`, `ramen promote`, and `ramen assess` rerun narrower stages after edits.
6. Quality gates validate project policy, OpenAPI availability, intent validity, data flow, workflow
   compilation, expected-plan matching, UWS profile/schema checks, review evidence, credential
   policy, side-effect policy, handoff contract, and secret scanning.
7. Reviewers inspect the minimum review package and, when appropriate, create approval JSON for
   sandbox or production tier.
8. `ramen run` revalidates the package, current quality, approval JSON, digest, tier/state rules,
   credential-value policy, and direct-production policy before writing run config and invoking the Go trusted executor runner by argv.

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

Ramen owns the structural progressive loop, draft/transcript lifecycle, prompt transcript/replay,
and JSON completion fallback plumbing under `internal/authoring`. `apitools` still supplies
prompt-safe OpenAPI operation context and ranking. Ramen's `internal/icot/elicitor/` files remain
the rollout binding for prompts, sanitization, readiness, final edit/explain confirmation, and
artifact rendering.

## Symphony And Trusted Execution

Ramen keeps `../symphony` untouched. The Ramen-owned handoff package gives Symphony or any reviewer
enough evidence to route review, but Ramen does not implement managed reviewer identity or audit
history.

The local trusted runner is intentionally separate from synthesis. It validates
`expected/symphony-handoff.json`, `expected/quality.json`, current in-memory quality, approval JSON,
canonical package digest, and tier compatibility. The package digest uses Ramen-local handoff digest
helpers over Ramen's required input set, including every regular file under `openapi/`.
`internal/packageartifacts` owns the required package inventory, safe relative path validation,
manifest-required path normalization, regular-file checks, digest input construction, and staging
input construction. Symlinked, directory, special-file, unsafe relative, and unstated required
handoff inputs are rejected before approval can authorize execution. It rejects credential values
in artifacts and direct production execution, then writes a non-secret `ramen.executor-run.v1`
config. The executor shim stages the reviewed UWS/OpenAPI files into the run workdir and invokes
udon through a binary or Docker process boundary, never through Go imports. Docker execution passes
only declared `UDON_CREDENTIAL_*` environment variable names into the container.

Required handoff inputs are `project.md`, `workflows/intent.hcl`, `workflows/workflow.hcl`,
`workflows/workflow.uws.yaml`, `expected/plan.json`, `expected/quality.json`,
`expected/refinement.json`, `expected/review.md`, `expected/symphony-handoff.json`, and any
`openapi/...` file staged for execution. Approval states are `generated`, `validated`,
`review_required`, `approved_for_sandbox`,
`approved_for_production`, and `rejected`.

Automation tiers:

| Tier | Gate | Location |
| --- | --- | --- |
| Local deterministic | `go test ./...`, `go vet ./...`, `make check`, `git diff --check` | Trusted workstation with public Ramen siblings. |
| Local readiness report | `ramen readiness --run-gates --out eval/readiness/local.json` | Trusted workstation with public Ramen siblings. |
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
- `internal/workflowintent/`: Ramen compatibility adapter over local authoring concepts.
- `examples/`: committed examples and eval corpus.
- `templates/`: project brief starter templates.
- `memory-bank/`: living project memory.
- `evolution/`: versioned prompt/result snapshots for milestone-level direction changes.

## Security Boundary

Generated UWS, OpenAPI, HCL, review, and approval artifacts are untrusted until validated. Ramen
must not put secrets in prompts, examples, eval fixtures, committed artifacts, or logs. Credential
bindings are symbolic names only until a trusted runtime resolves them. Production side effects are
never allowed from agent sessions, synthesis, build, promote, assess, iCoT, or eval.
