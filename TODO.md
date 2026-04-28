# Ramen Product Hardening TODO

This is the live post-POC hardening roadmap for making Ramen a stable internal product for
generating, validating, reviewing, and handing off workflow artifacts. `ideas.md` remains historical
implementation context; this file tracks current product-readiness work.

## Status Markers

- `[todo]` not started
- `[in-progress]` actively being implemented
- `[blocked]` waiting on an external dependency or decision
- `[done]` implemented and verified

## Current State

Ramen has moved beyond a pure proof of concept: the eval harness exists, ten eval examples have a
known-good real-LLM legacy-extraction baseline, deterministic quality gates validate generated
artifacts, bounded refinement is recorded, and secret scanning has been hardened against
workflow-reference false positives.

The remaining gap is product readiness. The current evidence is still narrow, real-provider runs can
vary, Gemini structured-output smoke reached zero legacy fallback with the original ten-example
baseline corpus, and expanded-corpus release evidence is tracked by XRD-009.

## [done] Post-POC Baseline

Goal: preserve the known-good baseline before broadening capability.

Why it matters: future changes need a stable comparison point and a clear record of what already
works.

Done when: the committed docs, eval reports, and quality checks describe the current system without
overstating product readiness.

- Eval harness covers ten examples under `examples/eval`.
- Real LLM smoke on 2026-04-28 passed all ten examples with `gemini-2.5-flash`.
- Refinement reports record prompt version, attempts, mode, failure class, and prompt snapshot.
- Quality gates cover project policy, OpenAPI availability, intent validity, workflow compilation,
  expected-plan matching, UWS validation, review evidence, and secret scanning.
- README documents deterministic checks versus optional real-LLM eval smoke tests.

## [in-progress] Eval Corpus Expansion

Goal: grow the eval set from smoke coverage to representative workflow coverage.

Why it matters: six passing examples prove viability, not robustness across the workflow space.

Done when: the corpus has 25-50 curated briefs with stable references and meaningful failure
coverage.

- Add examples for branching, conditionals, retries, fallbacks, and explicit error paths.
- Add API-heavy examples for auth schemes, multi-service chains, pagination variants, request bodies,
  response extraction, and write operations.
- Add negative examples for missing OpenAPI capability, missing credential policy, disallowed
  runtimes, unsafe side effects, and incomplete project briefs.
- Add runtime-only examples for `fnct`, approved `cmd`, denied `cmd`, and future profiles.
- Record per-example purpose, expected quality gates, and whether reference comparison is strict or
  advisory.

Slice 1 adds four high-signal pass fixtures before scaling further: `customer-export-two-pages`,
`crm-note-write`, `inventory-api-key-binding`, and `support-priority-routing`. The routing fixture
keeps branch selection inside an approved function adapter for now; later slices should add stricter
condition/switch fixtures after the harness can classify those failures cleanly. Continue toward
25-50 total briefs after these prove stable.

Slice 2 adds XRD-004 OpenAPI coverage fixtures: `cursor-pagination-report` for cursor pagination,
bearer security, and response cursor extraction; and `order-fulfillment-chain` for per-step
multi-service OpenAPI selection, request-body construction, response extraction, security schemes,
and a sandbox write operation. The coverage plan is documented in
`docs/xrd-004-openapi-eval-plan.md`.

## [done] Golden Reference Discipline

Goal: make reference issues actionable instead of merely informational.

Why it matters: the latest full real-LLM run passed quality gates but still produced reference
issues in some briefs; the team needs to know which differences matter.

Done when: eval reports distinguish acceptable naming drift from real behavioral regression.

- Classify reference issues as `advisory`, `warning`, or `blocking` per eval brief.
- Decide whether step names and output names are golden requirements or semantic hints.
- Add documented thresholds for allowable reference issues in local evals and future release runs.
- Add triage notes to reference fixtures when an illustrative reference is intentionally not exact.
- Track regressions against the previous run for pass rate, attempts, failure class, and blocking
  reference issues.

Slice 1 classifies reference drift as `advisory`, `warning`, or `blocking`, reports A/W/B counts in
eval Markdown, and treats increased blocking reference drift as an eval regression. Later slices
should add per-fixture policy/triage notes and release thresholds.

Slice 2 adds optional `reference/policy.json` files so each eval fixture can declare strict or
advisory reference comparison, issue triage notes, severity overrides, and per-fixture blocking
thresholds for release checks. Every current eval fixture has a policy file with `max_blocking: 0`;
`support-priority-routing` is marked advisory while its routing reference remains illustrative.
Step/output/request/bind names are documented as semantic hints, while runtime type, selected
OpenAPI operation, and reference parse/compare failures are behavioral drift.

## [done] Structured Output By Default

Goal: make provider-native structured generation the normal path and legacy extraction the fallback.

Why it matters: every latest real-LLM eval used legacy extraction; that keeps JSON parsing fragility
in the happy path.

Done when: supported providers use structured output by default and eval reports show zero legacy
fallbacks for the normal Gemini Flash run.

- Verify `rollout.StructuredChat` support and schema compatibility across Gemini, OpenAI, and
  Anthropic paths.
- Make eval reports fail or warn when fallback usage increases unexpectedly.
- Keep legacy extraction available only for unsupported clients and explicit compatibility tests.
- Add provider tests that assert schema, temperature, response mode, and fallback behavior.
- Record structured versus legacy mode clearly in refinement and eval summaries.

Slice 1 fixed Gemini structured-output request wiring in `../udon`, added fallback-regression
detection to eval comparison, tightened structured-mode prompt and intent cleanup for model-only
noise, and recovered the 2026-04-28 full `gemini-2.5-flash` run to 10/10 pass with `0` legacy
fallbacks.

## [done] Quality Gate Hardening

Goal: strengthen deterministic validation so generated artifacts are auditable before execution.

Why it matters: product trust should come from checks, not model optimism.

Done when: common artifact mistakes are caught with precise failure codes and actionable repair
guidance.

- Tighten data-flow checks for missing dependencies, ambiguous sources, wrong response paths, and
  undeclared function inputs.
- Harden credential checks around binding declarations, OpenAPI security schemes, request placement,
  and secret-value leakage.
- Expand side-effect checks for write operations, customer communications, command execution, and
  production endpoint usage.
- Improve review evidence so it records skipped execution, inferred technical steps, unresolved
  risks, and trusted-runner handoff requirements.
- Keep scanner tests populated only with fake secret-shaped values and valid workflow references.

Slice 1 adds deterministic side-effect policy checks and hardened review evidence. Review artifacts
now record side-effect summary, approval/trusted-runtime policy, sandbox/test proof-run policy,
unresolved risks, skipped execution, and trusted-runner handoff. Quality fails side-effectful
workflows that lack approval or trusted-runtime policy.

Slice 2 adds deterministic function-contract checks. Any generated `fnct` step must have a matching
project Function Contracts entry, projects that declare no function steps cannot generate `fnct`
steps, declared function inputs need visible `with`, `bind`, or prior-step evidence, and simple
contract input names reject undeclared adapter inputs.

Slice 3 completes the remaining deterministic quality gates. Intent assessment now fails unresolved
data-flow sources, definitely invalid OpenAPI response paths, missing OpenAPI security credential
policy, and unbound security schemes. Expected plans record OpenAPI security credentials so compiled
workflow checks catch misplaced auth fields. Side-effect assessment now detects write OpenAPI
methods, customer communication terms, command/SSH runtimes, and explicit production endpoints that
lack production handoff policy.

## [done] Workflow Artifact Power

Goal: support richer workflow artifacts without moving generic semantics out of `../uws` or
`../udon`.

Why it matters: post-POC workflows need more than linear API chains.

Done when: Ramen can generate and validate richer patterns using public UWS semantics and udon
compiler/runtime support.

- Add artifact generation coverage for conditionals, loops, pagination, retries, timeouts,
  idempotency keys, and failure branches where supported upstream.
- Improve function-contract handling for trusted adapters, transformations, renderers, and future
  runtime profiles.
- Represent hidden technical steps explicitly in intent, workflow, plan, review, and quality output.
- Keep product-specific policy in Ramen while pushing generic workflow or execution improvements to
  `../uws` or `../udon`.
- Add examples that prove generated artifacts remain readable and reviewable as workflows grow.

Slice 1 adds explicit switch/branch artifact support in the Ramen integration layer. Structured
intent now admits public UWS structural step types, the intent prompt includes a switch example, and
plan/review/quality output preserves parent, branch, branch condition, and control-flow fields for
nested steps. Quality coverage now indexes structural UWS steps as well as executable leaf
operations, so a planned switch step can be validated without pretending it is a leaf operation.

Slice 2 adds loop and structural-result compatibility in Ramen. Loop artifacts now preserve
`items`, `batch_size`, nested steps, plan evidence, review evidence, exported UWS, and quality
checks. Intent outputs that reference structural `switch`, `merge`, or `loop` steps are exported as
UWS `results[]`, and quality fails if exported results diverge from the expected plan. Retry and
failure-action compatibility is covered at the public UWS schema and udon execution-profile layer,
but Ramen still keeps retry/failure prompt defaults disabled until workflow draft lowering can carry
those action fields.

Slice 3 adds failure-action and retry compatibility for UWS-supported operation actions. Udon now
preserves `successCriteria`, `onFailure`, and `onSuccess` through rollout intent, workflow drafts,
canonical HCL, runtime plans, exec-cache conversion, program-view import/export, and UWS export.
Ramen intent schema, expected plans, plan Markdown, review evidence, and quality checks now preserve
and compare those action policies. Retry actions remain opt-in: prompts only emit them when the
project brief or intent explicitly asks, and side-effectful workflows with retry actions must include
explicit retry/idempotency policy in `project.md`.

## [done] Observability And Eval Analytics

Goal: make generation quality measurable over time.

Why it matters: model behavior, provider reliability, and prompt quality drift.

Done when: eval runs can be compared across prompt versions, models, providers, and commits.

- Track pass rate, attempts-to-pass, failure class, failing checks, mode, latency, model, provider,
  prompt version, and generated directory.
- Add approximate or provider-reported token and cost accounting where available.
- Preserve manual eval artifacts when a future workflow is reintroduced and make local eval output
  easy to compare.
- Add summaries for top failing checks and repeated repair loops.
- Document expected variance for real-LLM runs and avoid treating a single sample as proof of
  stability.

Slice 1 adds run-level eval analytics without changing real-provider execution. Eval JSON reports
now include a summary with pass/fail counts, pass rate, legacy fallback count, repeated repair-loop
count and brief names, approximate prompt-token total, duration totals/average/max, provider/model
/mode/prompt-version distributions, failure-class counts, and top failing checks. Per-brief results
now record total attempt count and whether a repeated repair loop occurred, while generated artifact
directories remain listed for manual inspection.

Slice 2 completes local eval comparability. Eval reports now include run metadata for commit,
dirty-worktree state, eval root, output path, provider, model, release-gate state, and run ID. Runs
compare against a selected or previous report by default and record pass-rate, brief, attempt,
legacy-fallback, blocking-reference, failing-check, duration, and prompt-token deltas in JSON and
Markdown. Comparison regressions are visible in normal eval output but fail only under
`--release-gate`. `--archive-dir` preserves generated eval workspaces for manual inspection, while
ignored `eval/runs/` and `eval/artifacts/` outputs keep real-provider artifacts out of commits.
Provider-reported token and cost fields are available when a provider path exposes usage data;
Ramen records approximate prompt tokens today and does not hardcode provider pricing.

## [done] Safety And Trusted Execution

Goal: make the approved path from generated artifact to trusted execution explicit.

Why it matters: Ramen may generate operational workflows, but production side effects must remain
behind review and approved runtime boundaries.

Done when: every side-effectful handoff has auditable approval evidence and credential binding
policy.

- Define the minimum review package for trusted execution: project brief, intent, workflow, UWS,
  plan, quality report, refinement report, and review evidence.
- Specify sandbox proof-run policy and when sandbox/test endpoints are required.
- Document credential binding audit rules and prohibit literal secrets in prompts, examples, and
  artifacts.
- Add checks for side-effectful workflows that lack approval, sandbox policy, or trusted runner
  handoff notes.
- Keep direct production execution out of agent-driven synthesis commands.

Slice 1 makes the trusted handoff package explicit. Review evidence now lists the minimum package
required for trusted execution, records credential binding audit requirements, states that Ramen
synthesis does not directly execute production workflows, and separates trusted-runner handoff notes
from validation evidence. Deterministic quality now requires side-effectful workflows to declare both
approval/trusted-runtime policy and sandbox/test proof-run policy before passing.

Slice 2 completes Ramen-owned trusted-execution evidence. Review artifacts now state that Ramen
emits only `generated` artifacts, side-effectful workflows require `review_required`,
`approved_for_sandbox`, and `approved_for_production` before the matching execution tier, and the
trusted-runner command is scoped to approved sandbox/proof execution. Review evidence also records
declared and expected credential binding names, or explicitly states that no credential bindings are
declared or required. Deterministic quality now fails missing approval-state, sandbox-handoff, or
credential-binding review evidence without adding any Symphony-owned approval routing.

## [done] Local Checks And Future Release Process

Goal: separate cheap deterministic development gates from expensive real-provider smoke tests.

Why it matters: regular development should be fast and stable; release confidence still needs real
model evidence.

Done when: local checks, manual evals, and future release checks have clear, repeatable gates.

- Keep `go test ./...`, `go vet ./...`, and `make check` as normal development gates.
- Keep GitHub workflows disabled until private sibling checkout and provider credential issues are
  stable enough to avoid noisy failures.
- Keep real-LLM evals local/manual for now, with optional uploaded artifacts only after workflows
  are reintroduced.
- Add release criteria for pass rate, structured-mode usage, maximum attempts, blocking reference
  issues, and zero secret-scan failures.
- Add a release note template that records model, prompt version, eval corpus size, pass rate, and
  known gaps.
- Decide whether `make eval` should default to `gemini-2.5-flash` to match README guidance.

Slice 1 aligns `make eval` with the README-backed `gemini-2.5-flash` default, adds an opt-in
`ramen eval --release-gate` for candidate release smoke runs, and adds
`docs/release-note-template.md`. The release gate requires a 100% pass rate, zero legacy extraction
fallbacks, no brief above two attempts, zero blocking reference issues, and zero
`artifacts.no_secrets` failures while keeping normal development checks deterministic and cheap.

Slice 2 closes the local operator release path. `make vet` exposes explicit vet parity,
`make release-check` runs deterministic release readiness (`go test ./...`, `go vet ./...`,
`make check`, and `git diff --check`), and `make release-eval` runs the opt-in manual
real-provider gate through `ramen eval --release-gate` using `RAMEN_PROVIDER` and `RAMEN_MODEL`.
The release note template now records comparison baseline, eval JSON/Markdown paths, commit and
dirty state, release-gate result, deterministic checks, and known external blockers while keeping
real-provider evals local/manual and GitHub workflows disabled.

## [done] Product Usability

Goal: make Ramen easier for trusted internal users to apply correctly.

Why it matters: product readiness depends on predictable operator experience, not only passing tests.

Done when: a new trusted user can author a project, run synthesis, interpret failures, and prepare a
handoff without reading implementation code.

- Improve CLI help for `synthesize`, `build`, `promote`, `assess`, and `eval` with examples and
  artifact descriptions.
- Add sample gallery notes explaining what each eval/example demonstrates.
- Improve error messages so failing quality checks point to the next concrete repair action.
- Keep `templates/project.md`, `docs/project-authoring.md`, and `docs/data-flow.md` aligned with
  prompt expectations.
- Add onboarding documentation for environment setup, required sibling repos, provider credentials,
  and real-LLM eval policy.

Slice 1 improves operator entry points. CLI help for artifact commands and eval now includes
examples and artifact descriptions, `assess` output includes next-action hints for failed quality
checks, `docs/onboarding.md` documents setup/credentials/eval policy, `docs/eval-gallery.md`
explains the current eval samples, and the project template plus authoring docs now reflect the
trusted-runner and sandbox proof-run safety requirements.

Slice 2 aligns the operator path with current release and safety behavior. Eval help now explains
normal comparison output versus release-gate failures, quality repair hints cover newer credential,
side-effect, approval-state, sandbox-handoff, trusted-runner, and production-boundary failures, and
`docs/operator-checklist.md` links project authoring, safety review, eval gallery, release notes,
and trusted handoff expectations.

## [blocked] Cross-Repo Dependencies

Goal: track hardening work that cannot be completed in Ramen alone.

Why it matters: Ramen should stay thin; generic semantics and execution support belong in sibling
projects.

Done when: each cross-repo dependency has an owner, target repo, and compatibility plan.

Execution sequencing, next artifacts, and follow-up plan ownership are tracked in
`docs/xrd-roadmap.md`. Keep this section as the status summary only.

Dependency status markers:

- `[blocked]` needs upstream work before Ramen can finish the capability.
- `[ready]` Ramen can add compatibility tests or integration glue now.
- `[handoff]` Ramen's handoff package is complete; an external owner owns implementation.
- `[watch]` external risk to monitor; no local code change is enough.
- `[done]` upstream capability exists; keep regression coverage in Ramen.

| ID | Status | Priority | Target | Capability | Ramen symptom/evidence | Owner | Compatibility plan |
| --- | --- | --- | --- | --- | --- | --- | --- |
| XRD-001 | `[done]` | P0 | `../udon` | Provider-native structured output for Gemini intent generation. | Structured eval smoke on 2026-04-28 reached 10/10 with zero legacy fallback after udon request wiring was fixed. | udon | Closed capability; `docs/xrd-roadmap.md` keeps regression ownership only. |
| XRD-002 | `[done]` | P0 | `../udon` | Preserve and lower public UWS structural constructs and failure actions from generated workflow drafts. | Workflow Artifact Power Slices 1-3 cover switch, loop, structural-result, success-criteria, failure-action, retry, and success-action artifact preservation through udon and Ramen compatibility checks. | udon | Closed capability; `docs/xrd-roadmap.md` keeps regression ownership only. |
| XRD-003 | `[done]` | P0 | `../uws` | Portable serialized timeout and workflow-level idempotency semantics not already in UWS 1.0. | UWS 1.1.0 defines `timeout` on operations, workflows, and steps, plus workflow-level `idempotency` metadata. Ramen generation behavior is unchanged in this pass. | uws | Cross-repo public semantics are closed; use a Ramen-owned follow-up if prompt/schema/eval support should emit UWS 1.1 fields. |
| XRD-004 | `[done]` | P1 | `../ramen`, then `../udon` | Generic OpenAPI execution/compiler behavior for richer API workflows. | Ramen now has a documented XRD-004 eval plan plus fixtures covering pagination variants, request bodies, security schemes, write operations, response extraction, and multi-service chains. | Ramen eval owner / udon | Keep `docs/xrd-004-openapi-eval-plan.md` and eval fixtures as the Ramen-owned coverage; propose reusable udon fixes only after concrete eval failures identify upstream gaps. |
| XRD-005 | `[handoff]` | P1 | External `../symphony` owner | Review workflow, approval handoff, and agent workspace policy integration. | Ramen emits the minimum handoff package, full approval-state review evidence, and `docs/xrd-005-symphony-handoff.md` for Symphony implementation. | symphony owner | Hand `docs/xrd-005-symphony-handoff.md` to the Symphony owner; contract details stay in `docs/cross-repo-contracts.md`. |
| XRD-006 | `[watch]` | P1 | Provider APIs | Structured-output schema dialect compatibility, rate limits, transient errors, and model availability. | Eval Markdown reports include a Provider Drift Watch section; `docs/xrd-006-provider-drift-watch.md` defines the release evidence path. | provider owners | Watch during release evals; no implementation plan is open. |
| XRD-007 | `[done]` | P1 | Repo access / secrets | Private sibling checkout and provider credential availability for future workflow automation. | Deterministic CI now checks out private siblings, sets `GOPRIVATE`, and uses owner-scoped CI tokens; real-LLM eval remains local/manual. | infra / Ramen | Maintain `docs/ci.md` and `.github/workflows/deterministic.yml`; provider-key automation remains out of CI. |
| XRD-008 | `[done]` | P2 | `../ramen`, then `../udon` / `../uws` | Runtime/profile coverage for approved non-HTTP execution beyond current `fnct`/`cmd` smoke paths. | Ramen has `docs/xrd-008-runtime-profile-eval-plan.md` plus fixtures covering approved `fnct`, approved `cmd`, denied `cmd`/`ssh`, and future profile-boundary behavior. | Ramen eval owner / udon/uws | Keep the Ramen coverage as regression evidence; open upstream udon/UWS work only when future fixtures need reusable runtime/profile semantics. |
| XRD-009 | `[done]` | P1 | `../ramen` | Real-provider release evidence for the expanded eval corpus beyond the original ten-example baseline. | `make release-eval` now passes a minimum brief count for the current corpus, and `docs/xrd-009-expanded-corpus-release-evidence.md` defines the evidence package. | Ramen release owner | Run local/manual release evals with the expanded-corpus gate; real-provider artifacts remain uncommitted. |

Next upstream actions:

1. For XRD-003, public UWS 1.1 contracts now exist; keep Ramen generation behavior unchanged until
   a Ramen-owned follow-up adds prompt/schema/eval coverage for emitting those fields.
2. For XRD-005, hand `docs/xrd-005-symphony-handoff.md` and the
   `docs/cross-repo-contracts.md` approval contract to the Symphony owner; do not modify
   `../symphony` from Ramen.
3. For XRD-007, keep CI tokens scoped to read-only private dependency access and keep provider keys
   out of deterministic CI.
4. Use `docs/xrd-roadmap.md` for the full XRD execution sequence and follow-up plan boundaries.
