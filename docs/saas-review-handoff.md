# SaaS Review And Trusted Handoff

Side-effectful SaaS packages are review artifacts until `openudon run` validates
approval and writes a trusted-runner config. Synthesis, build, promote, assess,
iCoT, and eval commands do not send messages, create records, upload files, or
call production endpoints.
For a provider-free release demo that exercises this boundary, see
[SaaS Operator Release Path](saas-operator-release.md).

## Review Evidence

For SaaS packages, `expected/review.md` should give a reviewer these sections:

- minimum review package with every digest-covered artifact;
- inferred steps and explicit cross-service bindings;
- side-effect summary and side-effect risk review;
- approval state requirements;
- approval artifact checklist;
- credential binding audit and credential scope matrix;
- unresolved risks;
- trusted-runner dry-run and proof-run commands.

The side-effect risk review names the step, service or runtime source, operation
or HTTP method/path when known, and the approval path required before execution.
The credential scope matrix keeps each symbolic binding associated with the step
and OpenAPI operation that uses it. Credential values must not appear in review
evidence, handoff manifests, approvals, run configs, logs, prompts, or examples.

## Approval JSON

Create approval JSON only after reviewing the current package and its digest:

```bash
mkdir -p approvals
go run ./cmd/openudon approval-template \
  --example ./examples/eval/order-fulfillment-chain \
  --state approved_for_sandbox \
  --reviewer "Reviewer Name" \
  > approvals/order-fulfillment-chain-sandbox.json
```

Approval JSON uses `openudon.approval.v1`. Required fields are `scope`, `state`,
`reviewer`, `approved_at`, and `package_sha256`; optional fields are
`expires_at` and `notes`. If any digest-covered file changes, discard the old
approval and create a new one after review.

Use `approved_for_sandbox` for sandbox or test-endpoint proof runs. Production
tier execution requires `approved_for_production` and an operator-controlled
trusted executor environment.

## Dry Run First

Run a dry run before allowing the executor to receive a config:

```bash
go run ./cmd/openudon run \
  --example ./examples/eval/order-fulfillment-chain \
  --tier sandbox \
  --approval approvals/order-fulfillment-chain-sandbox.json \
  --dry-run
```

The dry run validates the handoff manifest, stored and current quality reports,
approval scope, approval state, expiry, package digest, tier compatibility,
credential-value policy, and direct-production policy. It writes a non-secret
`openudon.executor-run.v1` config with package paths, `package_sha256`, tier,
workdir, and credential binding names.

Remove `--dry-run` only in a trusted operator environment with sandbox targets
or a production approval, reviewed credential bindings, and a configured
`OPENUDON_EXECUTOR`.

## Current Strict SaaS Fixtures

The strict multi-service fixtures that should preserve this evidence shape are:

- `itops-slack-jira-issue-intake`;
- `itops-incident-response-archive`;
- `order-fulfillment-chain`.

Supporting side-effect fixtures such as Gmail send, Slack post, webhook send,
Drive upload, and sandbox user/order create workflows should keep the same
approval, digest, credential, and trusted-handoff posture.
