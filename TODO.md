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

Ramen has moved beyond a pure proof of concept: the eval harness exists, ten eval examples pass with
a real LLM, deterministic quality gates validate generated artifacts, bounded refinement is recorded,
and secret scanning has been hardened against workflow-reference false positives.

The remaining gap is product readiness. The current evidence is still narrow, real-provider runs can
vary, all ten latest real-LLM evals used legacy JSON extraction, and reference drift is not yet
formally triaged.

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

## [todo] Eval Corpus Expansion

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

## [todo] Golden Reference Discipline

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

## [todo] Structured Output By Default

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

## [todo] Quality Gate Hardening

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

## [todo] Workflow Artifact Power

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

## [todo] Observability And Eval Analytics

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

## [todo] Safety And Trusted Execution

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

## [todo] Local Checks And Future Release Process

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

## [todo] Product Usability

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

## [blocked] Cross-Repo Dependencies

Goal: track hardening work that cannot be completed in Ramen alone.

Why it matters: Ramen should stay thin; generic semantics and execution support belong in sibling
projects.

Done when: each cross-repo dependency has an owner, target repo, and compatibility plan.

- `../udon`: provider-native structured generation, runtime/compiler support, profile validation,
  and generic OpenAPI/UWS execution improvements.
- `../uws`: public workflow semantics for any new conditionals, loops, retries, profiles, or
  execution metadata.
- `../symphony`: orchestration, work item policy, agent workspace handoff, and review workflow
  integration.
- Provider APIs: schema dialect compatibility, rate limits, transient errors, and model availability.
- Future workflow secrets and private sibling checkout permissions for manual eval workflows.
