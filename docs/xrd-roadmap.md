# XRD Coordination Roadmap

This is the umbrella coordination tracker for Ramen cross-repo dependencies. It sequences the XRD
items, assigns the next artifact, and names the follow-up plan that owns implementation work. It
does not add prompt behavior, schema fields, UWS semantics, udon runtime behavior, Symphony code, or
CI automation.

Detailed contracts remain in [`docs/cross-repo-contracts.md`](cross-repo-contracts.md). In
particular, that page is the source for XRD-003 public-semantics constraints, XRD-005 Symphony
handoff requirements, and XRD-007 private checkout and secrets prerequisites.

## Status Summary

| ID | Status | Owner | Target repo | Current status | Next artifact | Follow-up plan |
| --- | --- | --- | --- | --- | --- | --- |
| XRD-001 | Closed | udon / Ramen regression owner | `../udon` | Provider-native structured output for Gemini intent generation exists; Ramen structured eval smoke reached 10/10 with zero legacy fallback on 2026-04-28. | Regression report only if fallback behavior regresses. | None unless tests regress. |
| XRD-002 | Closed | udon / Ramen regression owner | `../udon` | Public UWS structural constructs and failure actions are preserved across udon and Ramen compatibility checks. | Regression report only if artifact preservation regresses. | None unless tests regress. |
| XRD-003 | Closed | uws owner | `../uws` | UWS 1.1.0 now defines portable `timeout` fields and workflow-level `idempotency` metadata in `../uws/versions/1.1.0.md` and `../uws/versions/1.1.0.json`. | Ramen follow-up only if prompt/schema/eval support should emit UWS 1.1 fields by default. | None for cross-repo public semantics. |
| XRD-004 | Closed | Ramen eval owner, then udon owner for reusable gaps | `../ramen`, then `../udon` | Ramen has `docs/xrd-004-openapi-eval-plan.md` plus fixtures covering pagination variants, request bodies, security schemes, write operations, response extraction, and multi-service chains. | Upstream udon issue only if future eval runs identify a reusable compiler/runtime gap. | None unless richer OpenAPI evals regress. |
| XRD-005 | Handed off | External Symphony owner | `../symphony` | Ramen emits review evidence, handoff files, and the concrete owner handoff in `docs/xrd-005-symphony-handoff.md`; approval routing remains external. | Symphony owner implementation from the documented handoff package and approval states. | None in Ramen unless the handoff contract changes. |
| XRD-006 | Watch | Ramen release owner / provider owners | Provider APIs | Provider behavior can drift in schema dialect support, rate limits, transient failures, and model availability. | Eval Markdown reports now include a Provider Drift Watch section, with the runbook in `docs/xrd-006-provider-drift-watch.md`. | No implementation plan; keep watching during release evals. |
| XRD-007 | Closed | Infra owner / Ramen | Repo access / secrets | Deterministic CI checks out private siblings and configures private module access with owner-scoped CI tokens; real-provider automation remains local/manual. | Regression report only if deterministic CI loses private dependency access. | None unless CI auth or checkout layout changes. |
| XRD-008 | Closed | Ramen eval owner, then udon/uws owners for reusable semantics | `../ramen`, then `../udon` / `../uws` | Ramen has `docs/xrd-008-runtime-profile-eval-plan.md` plus runtime/profile fixtures for approved `fnct`, approved `cmd`, denied `cmd`/`ssh`, and future profile-boundary behavior. | Upstream udon/UWS issue only if future fixtures require reusable runtime/profile semantics. | None unless runtime/profile evals regress. |
| XRD-009 | Closed | Ramen release owner | `../ramen` | Expanded eval corpus has a release-evidence gate beyond the original ten-example real-provider baseline. | `make release-eval` enforces a minimum brief count from the current eval corpus; evidence package lives in `docs/xrd-009-expanded-corpus-release-evidence.md`. | None unless release evidence is under-sampled or the corpus gate regresses. |

## XRD-001 Structured Output

Decision: closed capability with regression ownership only.

Acceptance criteria:

- Ramen names structured-mode eval reporting and legacy fallback counting as the regression coverage
  area.
- No new implementation plan is needed unless structured output falls back unexpectedly or eval
  reporting stops recording fallback behavior.

Implementation boundary: provider request wiring remains in `../udon`; Ramen keeps compatibility
checks and release evidence.

## XRD-002 UWS Artifact Preservation

Decision: closed capability with regression ownership only.

Acceptance criteria:

- Ramen names switch, loop, structural result, success-criteria, failure-action, retry, and
  success-action preservation as the regression coverage area.
- No new implementation plan is needed unless udon workflow, rollout, generator, exec-cache,
  program-view, UWS bridge, or Ramen synthesize quality tests regress.

Implementation boundary: generic lowering and execution remain in `../udon`; Ramen keeps artifact
quality and compatibility checks.

## XRD-003 UWS Public Semantics

Decision: closed upstream public-semantics handoff.

Next artifact: a Ramen-owned follow-up only if Ramen should emit UWS 1.1 `timeout` or
`idempotency` fields in generated artifacts.

Acceptance criteria:

- UWS defines portable timeout semantics in `../uws/versions/1.1.0.md`.
- UWS defines workflow-level idempotency metadata in `../uws/versions/1.1.0.md`.
- The compatibility matrix in
  [`docs/cross-repo-contracts.md#xrd-003-uws-public-semantics-audit`](cross-repo-contracts.md#xrd-003-uws-public-semantics-audit)
  distinguishes the completed public contract from future Ramen prompt/eval work.

Implementation boundary: Ramen may now plan prompt/schema/eval support against UWS 1.1, but this
pass does not change Ramen generation behavior.

## XRD-004 Richer OpenAPI Eval Coverage

Decision: closed Ramen follow-up plan with regression ownership.

Next artifact: upstream udon issue only if future eval runs identify a reusable compiler/runtime
gap.

Acceptance criteria:

- Eval coverage includes pagination, request bodies, security schemes, write operations, response
  extraction, and multi-service chains.
- Fixtures identify concrete compiler/runtime gaps before generic fixes are proposed in `../udon`.
- Product-specific behavior stays out of `../udon`.
- The coverage matrix lives in `docs/xrd-004-openapi-eval-plan.md`.

Implementation boundary: Ramen owns curated eval fixtures and policy evidence; `../udon` owns only
reusable OpenAPI/UWS compiler or runtime fixes found by those evals.

## XRD-005 Symphony Approval Handoff

Decision: Ramen handoff package complete; Symphony owner implementation remains external.

Next artifact: Symphony owner implementation using
[`docs/xrd-005-symphony-handoff.md`](xrd-005-symphony-handoff.md).

Acceptance criteria:

- The handoff uses the required files listed in
  [`docs/cross-repo-contracts.md#xrd-005-symphony-approval-handoff-contract`](cross-repo-contracts.md#xrd-005-symphony-approval-handoff-contract).
- The handoff uses the documented approval states from the same contract.
- Generated Ramen review evidence names the full state model: `generated`, `validated`,
  `review_required`, `approved_for_sandbox`, `approved_for_production`, and `rejected`.
- Ramen does not modify `../symphony`; it coordinates the upstream request with the Symphony owner.

Implementation boundary: Ramen owns deterministic artifact generation, validation, review evidence,
and trusted-runner command text. Symphony owns routing, reviewer identity, audit trail, state
transitions, workspace linkage, and production-execution enforcement.

## XRD-006 Provider Drift Watch

Decision: watch plan, not an implementation plan.

Next artifact: a provider drift watch report attached to release evaluation evidence when real
providers are used. The report source is the eval Markdown Provider Drift Watch section, documented
in [`docs/xrd-006-provider-drift-watch.md`](xrd-006-provider-drift-watch.md).

Acceptance criteria:

- Watch structured fallback count.
- Watch rate and transient failures.
- Watch model availability.
- Watch attempts-to-pass.
- Watch release-gate failures.
- Release notes include the provider drift watch findings when real-provider release evidence is
  used.

Implementation boundary: deterministic checks stay local and stable. Provider drift evidence informs
release decisions, but external provider variance is not treated as a Ramen-only implementation
bug.

## XRD-007 Infra Handoff

Decision: deterministic CI enabled with private dependency checkout and secret-scoped module auth.

Next artifact: regression report only if deterministic CI loses private dependency access.

Acceptance criteria:

- The handoff links the private sibling checkout prerequisites in
  [`docs/cross-repo-contracts.md#xrd-007-private-checkout-and-secrets-runbook`](cross-repo-contracts.md#xrd-007-private-checkout-and-secrets-runbook).
- The handoff links the self-hosted runner prerequisites in the same runbook.
- The handoff links the provider secret and artifact redaction prerequisites in the same runbook.
- The workflow records that provider keys are not available to deterministic CI.
- CI setup is documented in [`docs/ci.md`](ci.md).

Implementation boundary: Ramen owns deterministic CI checks and private checkout layout.
Infrastructure owns token provisioning, runner policy, and any future provider-key automation.

## XRD-008 Runtime/Profile Eval Coverage

Decision: closed Ramen follow-up plan with regression ownership.

Next artifact: upstream udon/UWS issue only if future fixtures need reusable runtime/profile
semantics beyond Ramen policy evidence.

Acceptance criteria:

- Ramen adds only policy and eval fixtures for runtime/profile coverage.
- Runtime/profile semantics remain upstream in `../udon` or `../uws`.
- Fixtures distinguish Ramen policy decisions from reusable execution or public semantics gaps.
- The coverage matrix lives in
  [`docs/xrd-008-runtime-profile-eval-plan.md`](xrd-008-runtime-profile-eval-plan.md).

Implementation boundary: Ramen owns project policy, curated fixtures, and compatibility evidence.
Profile semantics and generic execution belong upstream.

## XRD-009 Expanded Corpus Release Evidence

Decision: closed Ramen release-evidence gate with regression ownership.

Next artifact: local/manual real-provider release report when a candidate release is evaluated.

Acceptance criteria:

- `make release-eval` passes `--min-briefs` using the current eval corpus size.
- `ramen eval --release-gate --min-briefs N` fails when fewer than `N` briefs are run.
- Release notes record the eval corpus size, minimum brief gate, and any intentional exception.
- Real-provider eval outputs remain ignored and uncommitted.

Implementation boundary: Ramen owns release evidence, local checks, and ignored report paths.
Provider drift remains XRD-006, and automation/secrets remain XRD-007.
