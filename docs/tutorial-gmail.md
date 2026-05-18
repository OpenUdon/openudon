# Tutorial: Gmail Audit Receipt

This fixture models a sandbox Gmail-like send operation followed by a local audit receipt renderer.
It is side-effectful by shape, so the tutorial keeps execution language at the review, approval, and
sandbox handoff boundary.
For release evidence, prefer the ignored-workdir loop in
[SaaS Operator Release Path](saas-operator-release.md) so the demo stays
provider-free and does not modify checked-in fixture artifacts.

Fixture path:

```text
examples/eval/gmail-send-audit-receipt/project.md
examples/eval/gmail-send-audit-receipt/openapi/gmail.yaml
examples/eval/gmail-send-audit-receipt/reference/intent.hcl
```

The project brief has runtime inputs for recipient, subject, and message. The generated workflow
should pass those inputs to `send_message`, then bind the provider response fields into
`render_audit_receipt`.

## Run The Artifact Loop

```bash
go run ./cmd/openudon synthesize --example ./examples/eval/gmail-send-audit-receipt
go run ./cmd/openudon build --example ./examples/eval/gmail-send-audit-receipt
go run ./cmd/openudon assess --example ./examples/eval/gmail-send-audit-receipt
```

Review the generated files before approval:

```text
examples/eval/gmail-send-audit-receipt/expected/quality.md
examples/eval/gmail-send-audit-receipt/expected/review.md
examples/eval/gmail-send-audit-receipt/expected/symphony-handoff.json
```

## Approval Dry Run

Use sandbox approval only for proof-run readiness. Do not treat synthesis or build output as
permission to send email.

```bash
mkdir -p approvals
go run ./cmd/openudon approval-template \
  --example ./examples/eval/gmail-send-audit-receipt \
  --state approved_for_sandbox \
  --reviewer "Reviewer Name" \
  > approvals/gmail-send-audit-receipt-sandbox.json

go run ./cmd/openudon run \
  --example ./examples/eval/gmail-send-audit-receipt \
  --tier sandbox \
  --approval approvals/gmail-send-audit-receipt-sandbox.json \
  --dry-run
```

The dry run checks approval state, package digest, current and stored quality, and tier rules. It
does not invoke the executor.
