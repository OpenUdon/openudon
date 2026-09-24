# Retired milestone M52 - M52 - Ramen/OpenUdon Approval And Governance Contract Review

**Milestone.** M52
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M52.md
**Source status SHA-256.** 8e51fe2fc3b62d6398bb8444b51b58cfda7d5adc98a71a28b0275fe2cfb9542c
**Source milestone snapshot.** tabilet/docs/history/milestone-before-legacy-retirement.md.txt
**Snapshot SHA-256.** 26884eeda9af4ded30e6d545a33dca84c401d2320c22355fe4fb0336d5dcac71
**Evidence.** 71a4f78afbcf2180fc478ffa89c53544c9160648
**Worktree.** includes uncommitted changes
**Review.** not established
**Review iterations.** not recorded
**Verification.** Original status bytes and full earlier milestone bytes preserved by SHA-256; no fresh acceptance claim.
**Consolidated into.** Current milestone dashboard and maintained memory-bank guidance; full earlier text remains in the frozen snapshot.

## Status record

~~~~~~~~~~~~~~~~~~~~markdown
# M52 - Ramen/OpenUdon Approval And Governance Contract Review

## Goal

Compare the real Ramen and OpenUdon approval/governance contracts before
moving code or creating a shared module. M52 is contract review only: command
code, product logic, schemas, and module dependencies stay in their current
repos.

## Status

Complete for the contract-review slice. Implementation remains deferred to
Evidence M07 and later product adoption milestones.

## Task State

| Item | State | Notes |
|---|---|---|
| Ramen artifact inventory | `[+]` | Ramen owns `ramen.approval.v1` as a plan artifact digest over plan version, project/API/state/control inputs, governance, approvers, resources, and diagnostics. `ramen.governance.v1` owns policy decisions and approval requirements over desired-state resources or run targets. `ramen.run.v1` owns imperative UWS run approval digests, executor results, and SQLite run/event history. |
| OpenUdon artifact inventory | `[+]` | OpenUdon owns `openudon.approval.v1` as package/tier approval over scope, state, reviewer, time/expiry, and `package_sha256`; `apitools.review-handoff.v1` as review package state/policy/credential/trusted-runner metadata; `openudon.executor-run.v1` as staged trusted-runner config; and `openudon.run-evidence.v1` as package gate, staging, credential-binding, and executor-invocation evidence. |
| Common-core classification | `[+]` | Shared core is limited to neutral approval evidence already represented by `github.com/OpenUdon/evidence/approval`: subject, approver, requirement, decision, digest references, and normalization. Ramen resources/actions, governance policies, desired-state approval digests, OpenUdon tier/state/package policy, handoff manifests, and runner configs remain downstream-owned. |
| Async evidence boundary classification | `[+]` | Shared async evidence should live in Evidence only as neutral execution request/response/status/read-observation/attempt records. Runtime hints are execution metadata, not desired-state hash inputs. Ramen owns convergence interpretation; OpenUdon owns package/run forwarding and audit capture. |
| Package/run forwarding task | `[+]` | OpenUdon may later embed or forward Evidence M07 records in run evidence or handoff artifacts, but only as execution observations. It must not decide Ramen resource convergence, mutate Ramen state, or become the owner of Ramen executor contracts. |
| Versioning recommendation | `[+]` | Do not create a new shared governance module now. Reuse `evidence/approval` for neutral approval mechanics and add Evidence M07 for async record shapes. Keep product wire versions (`ramen.*`, `openudon.*`, `apitools.review-handoff.v1`) unchanged until implementation consumers justify explicit migrations. |
| Deferred sharing review | `[+]` | Executor interfaces, run configs, CLI helpers, package layouts, state/revision history, and convergence policies stay deferred and product-owned. Shared extraction should be limited to deterministic records and redaction/digest helpers with two real consumers. |
| Boundary documentation | `[+]` | `docs/async-operations.md` now treats Evidence as the neutral record-shape owner and product repos as embedding/policy owners, avoiding a false OpenUdon-as-format-owner dependency. |

## Review Findings

| Severity | Finding | Follow-up |
|---|---|---|
| Medium | The first async boundary proposal incorrectly implied OpenUdon owned the shared `execution_request` format and was the producer for Ramen consumption. Current code shows no OpenUdon-to-Ramen runner path; OpenUdon owns package/run handoff while Ramen owns its executor request and reconciliation path. | Corrected `docs/async-operations.md`; Evidence M07 must own neutral record shapes, and Ramen M29 must not assume OpenUdon is required in the apply/run path. |
| Low | Approval/governance overlap is already partially solved by `evidence/approval`; creating another shared governance module now would duplicate existing primitives before proving additional common behavior. | Keep M52 recommendation to reuse `evidence/approval` and limit new shared work to async evidence record envelopes. |

## Recommendation

Run the remaining dependency chain as:

1. Evidence M07 defines neutral async evidence record contracts.
2. OpenUdon adds package/run forwarding only after those records exist.
3. Ramen M29 consumes those records as reconciliation inputs, while preserving Ramen-owned convergence and state semantics.

## Post-M07 Forwarding Shape

Evidence M07 now provides `async.ExecutionRequest`,
`async.ExecutionResponse`, `async.StatusObservation`,
`async.ConfirmationReadObservation`, and `async.AttemptMetadata`.

OpenUdon follow-up adoption should be a later implementation milestone, not
part of M52. That milestone should decide whether `openudon.run-evidence.v1`
embeds a list of these neutral records, references sidecar files by digest, or
leaves them entirely in executor-owned output. Any option must preserve the
current OpenUdon responsibilities: package gates, approval/tier checks,
credential-binding evidence, trusted-runner invocation evidence, and no
desired-state convergence decisions.

## Verification

Completed checks:

```bash
go run ./cmd/openudon check-doc-memory
git -C ../tofu diff --check -- evidence openudon ramen
```
~~~~~~~~~~~~~~~~~~~~
