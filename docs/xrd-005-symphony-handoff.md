# XRD-005 Symphony Approval Handoff

This is the Ramen-owned handoff package for the external Symphony owner. It uses the public
`github.com/tabilet/apitools` review state machine and handoff schema, then adds Ramen-specific
artifact paths, Symphony owner split, and trusted-runner command text. It turns the approval contract in
[`docs/cross-repo-contracts.md#xrd-005-symphony-approval-handoff-contract`](cross-repo-contracts.md#xrd-005-symphony-approval-handoff-contract)
into an implementation request for Symphony work-item routing. Ramen does not modify
`../symphony` from this repository.

## Owner Split

| Area | Owner |
| --- | --- |
| Artifact generation and deterministic validation | Ramen |
| Review evidence, side-effect summary, credential-binding inventory, and trusted-runner command text | Ramen |
| Trusted local execution gate before udon invocation | Ramen |
| Reviewer identity, audit trail, and managed-state routing | Symphony |

## Handoff Inputs

Symphony should attach or link these Ramen outputs on the managed work item:

| Path | Required use |
| --- | --- |
| `project.md` | Source policy and safety boundary. |
| `workflows/intent.hcl` | Structured intent for review. |
| `workflows/workflow.hcl` | udon workflow source for trusted execution. |
| `workflows/workflow.uws.yaml` | Public UWS artifact validated by Ramen. |
| `expected/plan.json` | Machine-readable steps, bindings, credentials, control flow, and side effects. |
| `expected/quality.json` | Deterministic quality-gate result. |
| `expected/refinement.json` | Generation attempts, failed checks, and stop reason. |
| `expected/review.md` | Human review evidence and trusted-runner handoff text. |
| `expected/symphony-handoff.json` | Machine-readable apitools review handoff manifest with inputs, approval states, owner split, execution policy, credential bindings, and trusted-runner command. |

## Approval States

Symphony should use the state names below exactly so Ramen review evidence and Symphony routing stay
compatible.

| State | Meaning | Allowed next states |
| --- | --- | --- |
| `generated` | Ramen emitted artifacts; no approval is implied. | `validated`, `rejected` |
| `validated` | Required validators and quality gates passed, or known warnings are attached. | `review_required`, `rejected` |
| `review_required` | Human review is required before side-effectful execution. | `approved_for_sandbox`, `approved_for_production`, `rejected` |
| `approved_for_sandbox` | A reviewer approved sandbox or test-endpoint execution only. | `review_required`, `approved_for_production`, `rejected` |
| `approved_for_production` | A reviewer approved production execution through a trusted runner and approved credentials. | `rejected` |
| `rejected` | A reviewer rejected the artifact or requested regeneration. | `generated` after regeneration |

## Symphony Acceptance Criteria

- A Symphony work item can store the Ramen handoff inputs listed above.
- A Symphony workflow can route from `generated` through validation and review without implying
  execution approval.
- Reviewer identity and state-transition history are recorded outside Ramen artifacts.
- Side-effectful sandbox proof runs use `approved_for_sandbox` approval JSON.
- Production execution uses `approved_for_production` approval JSON.
- Trusted-runner commands are copied from `expected/review.md`; Symphony does not ask agents to
  execute production workflows directly.
- Credential values are never copied into Symphony prompts, review text, generated artifacts, or
  uploaded logs.

## Ramen Boundary

Ramen emits deterministic artifacts, quality reports, review evidence, and
`expected/symphony-handoff.json` for machine-readable work-item routing. Ramen also owns the local
trusted wrapper documented in `SYMPHONY_WRAPPER.md`:

```bash
go run ./cmd/ramen approval-template --example examples/support-email --state approved_for_sandbox --reviewer "Reviewer Name" > approvals/support-email-sandbox.json
go run ./cmd/ramen run --example examples/support-email --tier sandbox --approval approvals/support-email-sandbox.json
```

The wrapper validates the handoff package, current quality gates, approval file, package digest, and
tier/state compatibility before invoking `scripts/run-udon.sh`. It does not implement Symphony
reviewer assignment, audit logs, workspace policy, or managed state transitions in this repository.
