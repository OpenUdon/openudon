# Tutorial: Support Email

This fixture fetches a support ticket through OpenAPI and prepares a confirmation email through an
approved function adapter. It demonstrates the approval boundary for a side-effectful workflow.

Fixture path:

```text
examples/eval/support-email/project.md
examples/eval/support-email/openapi/support.yaml
examples/eval/support-email/reference/intent.hcl
examples/eval/support-email/reference/plan.json
examples/eval/support-email/reference/workflow.hcl
```

The project brief requires generated artifacts only. Any real email send must be routed through
human approval, sandbox proof-run policy, and trusted-runner handoff.

## Run The Artifact Loop

```bash
go run ./cmd/openudon synthesize --example ./examples/eval/support-email
go run ./cmd/openudon build --example ./examples/eval/support-email
go run ./cmd/openudon assess --example ./examples/eval/support-email
```

Inspect the side-effect evidence:

```text
examples/eval/support-email/expected/plan.md
examples/eval/support-email/expected/quality.md
examples/eval/support-email/expected/review.md
examples/eval/support-email/expected/review-handoff.json
```

The review evidence should describe the email side effect, credential bindings by name only, and
the sandbox or trusted-runtime boundary.

## Approval Dry Run

Generate sandbox approval from the current package digest and validate the handoff gates:

```bash
mkdir -p approvals
go run ./cmd/openudon approval-template \
  --example ./examples/eval/support-email \
  --state approved_for_sandbox \
  --reviewer "Reviewer Name" \
  > approvals/support-email-sandbox.json

go run ./cmd/openudon run \
  --example ./examples/eval/support-email \
  --tier sandbox \
  --approval approvals/support-email-sandbox.json \
  --dry-run
```

Remove `--dry-run` only in an operator-controlled environment with a reviewed executor, sandbox
targets, and approved credential bindings.
