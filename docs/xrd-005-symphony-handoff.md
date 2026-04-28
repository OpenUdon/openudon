# XRD-005 Symphony Approval Handoff

This is the Ramen-owned handoff package for the external Symphony owner. It turns the
approval contract in
[`docs/cross-repo-contracts.md#xrd-005-symphony-approval-handoff-contract`](cross-repo-contracts.md#xrd-005-symphony-approval-handoff-contract)
into an implementation request for Symphony work-item routing. Ramen does not modify
`../symphony` from this repository.

## Owner Split

| Area | Owner |
| --- | --- |
| Artifact generation and deterministic validation | Ramen |
| Review evidence, side-effect summary, credential-binding inventory, and trusted-runner command text | Ramen |
| Work-item routing, reviewer identity, audit trail, workspace linkage, and state transitions | Symphony |
| Enforcement that production execution cannot occur before the approved state | Symphony |

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
- Side-effectful sandbox proof runs are blocked until `approved_for_sandbox`.
- Production execution is blocked until `approved_for_production`.
- Trusted-runner commands are copied from `expected/review.md`; Symphony does not ask agents to
  execute production workflows directly.
- Credential values are never copied into Symphony prompts, review text, generated artifacts, or
  uploaded logs.

## Ramen Boundary

Ramen will keep emitting deterministic artifacts, quality reports, review evidence, and trusted-runner
command text. It will not implement Symphony reviewer assignment, audit logs, workspace policy, or
state-transition enforcement in this repository.
