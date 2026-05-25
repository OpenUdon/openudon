# OpenUdon Review Evidence

- Project brief: `project.md`
- Intent HCL: `workflows/intent.hcl`
- Workflow HCL: `workflows/workflow.hcl`
- UWS artifact: `workflows/workflow.uws.yaml`
- Expected plan: `expected/plan.json`
- Discovery report: `expected/discovery.json`
- Refinement report: `expected/refinement.json`
- Primary OpenAPI: `openapi/slack.yaml`

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

- `openapi/slack.yaml` - Slack Chat API (local)

## OpenAPI Discovery Attempts

- `local` pass `/home/peter/Workspace/openudon/examples/slack-message-audit-log/openapi` - 1 local OpenAPI document(s)

## Catalog Advisory

- Catalog metadata is advisory. Explicit OpenAPI inputs and human review remain authoritative for generated intent.
- Provider: `Slack` (`slack`)
  - Matched hint: `openapi/slack.yaml` from `intent.openapi`
  - Explicit OpenAPI input overrides built-in catalog spec: `openapi/slack.yaml`
  - Catalog spec: `slack-web-openapi-v2` `openapi` `https://raw.githubusercontent.com/slackapi/slack-api-specs/master/web-api/slack_web_openapi_v2_without_examples.json`
  - User OpenAPI need: `possible`
  - Auth/security status: `present-incomplete`
  - Security overlays: `slack-web-api-auth-review`
  - Source note: Slack's archived official OpenAPI document includes OAuth metadata and operation security but needs freshness review against current Slack docs.
  - Source note: Slack's official OpenAPI document includes OAuth metadata and operation security, but the repository is archived and lacks root security; keep this as a present-incomplete review overlay rather than treating it as current provider truth.

## Inferred Steps And Data Flow

- `post_message` (http) operation `postMessage`: Post one sandbox chat message.
- `render_audit_log` (fnct): Render a local audit log from the post response.
  - bind from `post_message`: `channel <- received_body.channel`: `ok <- received_body.ok`: `ts <- received_body.ts`

## Side-Effect Summary

- Side-effectful workflow: yes
- Evidence: post_message appears side-effectful
- Evidence: post_message uses POST /chat.postMessage
- Evidence: render_audit_log appears side-effectful
- Evidence: render_audit_log side effects: declared by approved function adapter; execute only through trusted runner approval.
- Approval/trusted-runtime policy: present in project.md.
- Sandbox/test proof-run policy: present in project.md.
- Credential binding audit: runtime binding names only; literal secrets are prohibited in prompts, examples, and artifacts.
- Direct production execution: not performed by OpenUdon synthesis.

## Side-Effect Risk Review

- `post_message` http operation `postMessage` from `intent language`: intent text contains create/send/write/update/delete/post style behavior
- `post_message` openapi operation `postMessage` `POST /chat.postMessage` from `openapi/slack.yaml`: write-class HTTP operation requires review and approved trusted-runner handoff
- `project policy` policy from `project.md`: project policy mentions side-effectful behavior
- `render_audit_log` fnct from `function contract`: declared by approved function adapter; execute only through trusted runner approval.
- `render_audit_log` fnct from `intent language`: intent text contains create/send/write/update/delete/post style behavior
- Required approval path: `review_required` -> `approved_for_sandbox` for sandbox/test proof runs; `approved_for_production` for production.

## Approval State Requirements

- OpenUdon emitted state: `generated`; no approval is implied by artifact generation.
- `validated`: required validators and quality gates have passed or known warnings are attached.
- `review_required`: human review is required before side-effectful execution.
- `approved_for_sandbox`: sandbox or test-endpoint execution only.
- `approved_for_production`: production execution through a trusted runner and approved credentials.
- `rejected`: artifact rejected or regeneration requested.
- Required next state: `review_required` before any side-effectful execution.
- Sandbox proof run requires review state `approved_for_sandbox`.
- Production execution requires review state `approved_for_production` and trusted credentials.
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
- Trusted proof run command is for sandbox/test execution only after `approved_for_sandbox`.
- Production execution requires `approved_for_production`; do not use this command as production approval.
- Credential binding audit must verify named runtime bindings and no literal secret values.
- Dry-run handoff validates approval state, package digest, stored/current quality, tier compatibility, credential-value policy, and direct-production policy before executor invocation.
- The generated run config is `openudon.executor-run.v1`; it carries package paths, `package_sha256`, tier, workdir, and credential binding names, not credential values.

Trusted dry run, before any executor invocation:

```bash
openudon run --example slack-message-audit-log --tier sandbox --approval approvals/slack-message-audit-log.json --dry-run
```

Trusted proof run, only when explicitly approved:

```bash
openudon run --example slack-message-audit-log --tier sandbox --approval approvals/slack-message-audit-log.json
```
