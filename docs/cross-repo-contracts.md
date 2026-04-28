# Cross-Repo Contracts

This page records the planning contracts for Ramen blockers that cannot be completed inside Ramen
alone. Ramen stays an integration layer: public workflow semantics belong in `../uws`, generic
compilation and execution belong in `../udon`, Symphony workflow ownership belongs with the external
`../symphony` owner, and private checkout or secret automation belongs to infrastructure.

## XRD-003 UWS Public Semantics Audit

Ramen must not teach prompts to emit workflow semantics that lack a public UWS contract. If a
capability is covered by UWS but not yet proven through udon and Ramen fixtures, keep it out of
default generation until compatibility is demonstrated.

The original UWS-public-semantics blocker was intentionally narrow: portable serialized timeout
fields and workflow-level idempotency metadata were not UWS 1.0 core fields. UWS 1.1 now defines
those public fields in `../uws/versions/1.1.0.md` and `../uws/versions/1.1.0.json`. Loops,
structural results, failure branches, retries, and runtime profiles are not blocked on `../uws`;
they need Ramen/udon compatibility proof before Ramen expands prompt defaults.

| Capability | Already covered by UWS 1.0 | Belongs in `../uws` | Belongs in `../udon` | Ramen policy only | May Ramen generate today |
| --- | --- | --- | --- | --- | --- |
| Switch branches | Yes: `type: switch`, `cases`, `when`, and `default`. | No new public semantics. | Preserve, lower, and execute generated switch constructs. | Decide when a project should use a switch. | Yes. Slice 1 prompt/plan/review/quality support exists. |
| Loops | Yes: `type: loop`, `items`, optional `batchSize`, and iteration context. | No new public semantics unless new loop modes are needed. | Preserve, lower, and execute loop constructs from Ramen-generated drafts. | Decide whether batch processing is allowed for the project. | Yes. Slice 2 proves loop draft lowering, plan/review evidence, UWS export, and quality coverage. |
| Structural results | Yes: top-level `results[]` with `kind` `switch`, `merge`, or `loop` and `from`. | No new public semantics for current result shapes. | Preserve structural result declarations and expose runtime outputs. | Require readable review evidence for result meaning. | Yes for generated structural step outputs. Slice 2 exports and validates UWS `results[]` from intent outputs that reference `switch`, `merge`, or `loop` steps. |
| Failure branches | Yes: `onFailure` actions support `end`, `goto`, and `retry` with criteria. | No new public semantics for basic failure routing. | Preserve and execute failure actions, including target resolution and retry counting. | Require side-effect review notes for failure paths. | Yes, when the brief or intent explicitly asks for failure routing. Slice 3 proves draft lowering, plan/review evidence, UWS export, and quality coverage. |
| Retries | Yes: `onFailure` action `type: retry`, `retryLimit`, and optional `retryAfter`. | No new public semantics for bounded failure retries. | Lower and run retry behavior reliably. | Limit retries for side-effectful operations and require idempotency review. | Yes, when the brief or intent explicitly asks for retry. Side-effectful retries require explicit retry/idempotency policy in `project.md`. |
| Timeouts | Yes in UWS 1.1: `timeout` is available on operations, workflows, and steps. | Done for the portable field contract; future spec changes only if semantics need revision. | Own executor options or profile-specific timeout behavior and runtime enforcement. | Decide when project policy should request portable timeouts. | Not by default in Ramen yet; allowed only after a Ramen follow-up adds prompt/schema/eval coverage for UWS 1.1 timeout. |
| Idempotency keys | Yes in UWS 1.1 for workflow-level `idempotency`; API-specific keys can still be ordinary OpenAPI request headers/body fields. | Done for the portable metadata contract; future spec changes only if semantics need revision. | Own generic runtime support for automatic key injection, replay protection, or idempotency record storage. | Require side-effect review evidence and credential-binding/idempotency policy. | API-specific request bindings may be generated today; workflow-level UWS 1.1 idempotency needs a Ramen follow-up before prompt defaults emit it. |
| Runtime profiles | Yes: extension-owned operations must declare `x-uws-operation-profile`; profile payloads are extensions. | Only profile marker semantics and reserved prefix policy. | Define and execute `x-udon-*` profile behavior. | Allow or deny profiles by project/environment policy. | Yes for existing validated udon profile shapes already covered by Ramen checks. |

Decision rule: when the table says "No", Ramen may document the need and validate existing
artifacts, but it must not add prompt defaults that synthesize the capability. When the table says
"Yes, but only", Ramen may generate that capability only in the named constrained form. UWS-covered
capabilities with "No" in the final column are ready compatibility work, not public-semantics
blockers.

## XRD-005 Symphony Approval Handoff Contract

Ramen emits an artifact package that Symphony can attach to a work item and route through review.
The package is deterministic file output, not an approval workflow by itself.
The concrete implementation request for the external owner is
[`docs/xrd-005-symphony-handoff.md`](xrd-005-symphony-handoff.md).

Required handoff package:

| Path | Purpose |
| --- | --- |
| `project.md` | Source brief, integration policy, runtime policy, credentials policy, safety boundary, and fallback behavior. |
| `workflows/intent.hcl` | Structured intent extracted from the project brief. |
| `workflows/workflow.hcl` | udon workflow source produced from intent. |
| `workflows/workflow.uws.yaml` | Exported UWS artifact validated against the public UWS schema and udon profile checks. |
| `expected/plan.json` | Machine-readable expected steps, bindings, credentials, control flow, and side-effect hints. |
| `expected/quality.json` | Deterministic quality gate results. |
| `expected/refinement.json` | Generation/refinement attempts, failed checks, and stop reason. |
| `expected/review.md` | Human review evidence, unresolved risks, skipped execution notes, and trusted-runner command text. |

Required approval states for the Symphony-owned work item:

| State | Meaning |
| --- | --- |
| `generated` | Ramen has emitted the handoff package. No approval is implied. |
| `validated` | Required validators and quality gates have passed or known warnings are attached. |
| `review_required` | Human review is required before any side-effectful execution. |
| `approved_for_sandbox` | A reviewer approved sandbox or test-endpoint execution only. |
| `approved_for_production` | A reviewer approved production execution through a trusted runner and approved credentials. |
| `rejected` | A reviewer rejected the artifact or requested regeneration. |

Ramen remains responsible for:

- Artifact generation and deterministic validation.
- Review evidence, including side-effect summary, unresolved risks, skipped execution, and sandbox
  proof-run requirements.
- Trusted-runner command text that an approved operator can execute outside the agent session.
- Secret scanning and credential-binding evidence using names, not secret values.

The external Symphony owner owns:

- Approval routing, reviewer identity, and audit trail.
- State transitions between the approval states above.
- Workspace and work item linkage.
- Enforcement that production execution cannot occur from an unapproved state.

Acceptance: a Symphony implementer can consume the listed files, map them to the listed states, and
build reviewer routing without guessing what Ramen emits or which approval state names are expected.
Ramen must not modify `../symphony`; it only maintains this emitted-artifact contract and coordinates
the upstream request with the Symphony owner.

## XRD-007 Private Checkout And Secrets Runbook

Deterministic GitHub CI is available for this private workspace, but it must use private dependency
checkout and private module authentication.
The concrete infrastructure handoff is
[`docs/xrd-007-infra-handoff.md`](xrd-007-infra-handoff.md).
Workflow setup details are in [`docs/ci.md`](ci.md).

Current blockers:

- Ramen imports private udon packages, and local Go builds need private sibling repos:
  `../udon`, `../grand`, `../golet`, `../hcllight`, `../horizon`, `../molecule`, and `../arazzo`.
- Real synthesis and eval need provider credentials.
- Real LLM results can vary by provider availability, model behavior, and transient failures.
- Secret exposure risk is higher in hosted logs, prompts, generated artifacts, and uploaded eval
  bundles.

Automation tiers:

| Tier | Gate | Where it runs | Notes |
| --- | --- | --- | --- |
| Local deterministic | `go test ./...`, `go vet ./...`, `make check` | Developer workstation with private siblings checked out. | Normal development gate. |
| Local/manual real LLM | `go run ./cmd/ramen eval --root ./examples/eval --provider <provider> --model <model> --release-gate` | Trusted workstation with provider credentials in environment variables. | Release smoke gate; results are reviewed manually. |
| CI deterministic | `go test ./...`, `go vet ./...`, `make check` | GitHub Actions private checkout with `RAMEN_CI_GENELET_TOKEN` and `RAMEN_CI_TABILET_TOKEN`. | No provider keys; no generated artifact uploads. |
| Future manual/release real-provider workflow | Release-gated eval command with explicit provider/model. | Protected manual workflow or trusted release machine. | Requires secret store controls and log/artifact redaction policy. |

Allowed secret handling:

- Provider API keys may exist only in a CI secret store or trusted local environment variables.
- Prompts, examples, generated OpenAPI/UWS/HCL artifacts, review files, eval fixtures, and logs must
  not contain literal secrets.
- Generated artifacts should refer to credential binding names only.
- CI logs and uploaded artifacts must be redacted or disabled for any command that may include model
  prompts, provider responses, or generated side-effect configuration.

Prerequisites for deterministic CI:

- `RAMEN_CI_GENELET_TOKEN` has read access to private `genelet/*` dependency repos.
- `RAMEN_CI_TABILET_TOKEN` has read access to private `tabilet/*` dependency repos.
- Private repository preflight passes for each required checkout and transitive private module repo.
- Private sibling checkout works non-interactively for every required repo.
- CI has no provider keys for deterministic checks.
- `go test ./...`, `go vet ./...`, and `make check` pass on a clean private checkout.
- Logs and artifacts exclude generated files that could contain prompt or credential-binding detail.

Prerequisites to add a real-provider release workflow:

- Provider keys are stored only in the protected secret store.
- The workflow is manual or release-only, never a routine PR gate.
- Eval output records provider, model, prompt version, generated directory, pass rate,
  attempts-to-pass, fallback count, and blocking reference issues.
- Uploaded artifacts are reviewed for secret redaction before retention is enabled.

Acceptance: infrastructure can maintain CI by checking these prerequisites without inferring private
repo layout, secret rules, or which checks are allowed in each tier.
