# OpenUdon Review Evidence

- Project brief: `project.md`
- Intent HCL: `workflows/intent.hcl`
- Workflow HCL: `workflows/workflow.hcl`
- UWS artifact: `workflows/workflow.uws.yaml`
- Expected plan: `expected/plan.json`
- Discovery report: `expected/discovery.json`
- Refinement report: `expected/refinement.json`
- Primary OpenAPI: ``

## Minimum Review Package

- Project brief: `project.md`
- Intent HCL: `workflows/intent.hcl`
- Workflow HCL: `workflows/workflow.hcl`
- UWS artifact: `workflows/workflow.uws.yaml`
- Expected plan: `expected/plan.json`
- Quality report: `expected/quality.json`
- Refinement report: `expected/refinement.json`
- Review evidence: `expected/review.md`
- Review handoff manifest: `expected/review-handoff.json`
- OpenUdon review package: `13` artifact(s), `0` symbolic binding(s), execution deferred to OpenUdon trusted-runtime policy.

## OpenAPI Candidates


## OpenAPI Discovery Attempts

- `local` pass `/home/peter/Workspace/openudon/examples/eval/openrpc-simple-math-addition/openapi` - OpenAPI directory is absent; using package-local first-class API source documents.

## Inferred Steps And Data Flow

- `add_numbers` (http) operation `addition`: Run the simple-math addition method.

## Side-Effect Summary

- Side-effectful workflow: no side-effectful behavior inferred from project policy or intent steps.
- Approval/trusted-runtime policy: present in project.md.
- Sandbox/test proof-run policy: not detected in project.md.
- Credential binding audit: runtime binding names only; literal secrets are prohibited in prompts, examples, and artifacts.
- Direct production execution: not performed by OpenUdon synthesis.

## Side-Effect Risk Review

- No side-effectful operations were inferred for this package.

## Approval State Requirements

- OpenUdon emitted state: `generated`; no approval is implied by artifact generation.
- `validated`: required validators and quality gates have passed or known warnings are attached.
- `review_required`: human review is required before side-effectful execution.
- `approved_for_sandbox`: sandbox or test-endpoint execution only.
- `approved_for_production`: production execution through a trusted runner and approved credentials.
- `rejected`: artifact rejected or regeneration requested.
- `approved_for_sandbox` and `approved_for_production` are not required unless future changes add side effects.
- Approval artifact: create `openudon.approval.v1` JSON with `openudon approval-template` only after reviewing the current digest-covered package.

## Approval Artifact Checklist

- Approval JSON version: `openudon.approval.v1`.
- Required fields: `scope`, `state`, `reviewer`, `approved_at`, `package_sha256`.
- Optional fields: `expires_at`, `notes`.
- Sandbox tier accepts `approved_for_sandbox` or `approved_for_production`; production tier requires `approved_for_production`.
- `package_sha256` must match the current handoff package digest at `openudon run` time.
- Regenerate approval JSON after any digest-covered package file changes.

## Credential Binding Audit

- No credential bindings declared or required.
- Credential values must stay outside prompts, examples, generated artifacts, and logs.

## Credential Scope Matrix

- No credential bindings are declared or expected from the plan.
- Credential values: not allowed in generated artifacts.

## Unresolved Risks

- No unresolved execution-boundary risks detected by deterministic review.

## Validation

- Generated intent.hcl from project.md.
- Generated workflow.hcl as a public UWS document from OpenUdon intent.
- Exported workflow.uws.yaml and validated it against the UWS schema and local execution-profile checks.
- Side-effectful execution was skipped.

## Trusted Execution Handoff

- Direct production execution: not performed by OpenUdon synthesis.
- Human approval and trusted-runner invocation are required before operational side effects.
- Sandbox/test proof run is optional unless future changes add side effects.
- Credential binding audit must verify named runtime bindings and no literal secret values.
- Dry-run handoff validates approval state, package digest, stored/current quality, tier compatibility, credential-value policy, and direct-production policy before executor invocation.
- The generated run config is `openudon.executor-run.v1`; it carries package paths, `package_sha256`, tier, workdir, and credential binding names, not credential values.

Trusted dry run, before any executor invocation:

```bash
openudon run --example openrpc-simple-math-addition --tier sandbox --approval approvals/openrpc-simple-math-addition.json --dry-run
```

Trusted proof run, only when explicitly approved:

```bash
openudon run --example openrpc-simple-math-addition --tier sandbox --approval approvals/openrpc-simple-math-addition.json
```
