# Retired milestone M90 - Per-step executor outcomes in run evidence

**Milestone.** M90
**Outcome.** completed
**Retired.** 2026-09-30
**Source status.** tabilet/memory-bank/status-M90.md
**Source specification.** tabilet/memory-bank/milestone.md#m90---per-step-executor-outcomes-in-run-evidence
**Evidence.** a388b235edc733fd23962ff006d2406d4965b48f
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 2
**Verification.** make check, go vet ./..., focused and real-qualification race checks, make report-v5-qualification at clean accepted source ed5b206a524e6e193123b2d06714b75160379560, doc-memory guard and git diff --check passed; source publication independently verified; detailed logs and exact frozen build closure retained below.
**Consolidated into.** product.md, architecture.md, tech-stack.md, lessons.md, docs/per-step-run-evidence.md and evolution v46; Kinet W07 reconciled in db35053f52fb7ee0e81d833752829b83f76ddd96.

## Milestone specification

````markdown
### M90 — Per-step executor outcomes in run evidence

Carry per-step executor outcomes into `openudon run` evidence. Accept
`udon.execution-report.v5` from the published Udon M44 schema alongside
v2–v4, and request v5 from the executor only when explicitly configured, so
existing consumers are unchanged. Validate every per-step record against the
staged package plan (known step and operation IDs, bounded count and size)
and reject unknown or duplicate steps. Record per-step outcomes (`not_started`,
`succeeded`, `failed`, `unknown`), timestamps, and non-secret failure codes in
`openudon.run-evidence.v3`, selected only with explicit report v5. The chosen
contract is [per-step-run-evidence.md](../../docs/per-step-run-evidence.md). A missing report marks every step
`unknown`; an incomplete report is preserved as incomplete. Publish
conformance fixtures for a dry run, success, failure before the write step,
and an unknown write step. Qualify once, by explicit opt-in, against a real
executor built from the published Udon M44 revision and a loopback read+write
fixture (success, failed read, killed during the write). The scope is local
and provider-free: no deployment, live account, or target operation. This
partially promotes the "Trusted-runner capability expansion" candidate below.
**Approved reconciliation.** Choose the run-evidence version and explicit report-v5 selection contract
before consumer integration. Validate exact run/workflow identity, complete
supported inventory, operation/invocation identities and timestamp/outcome
consistency, with bounded records. Missing, malformed, mismatched or incomplete
inventory never implies not_started; preserve uncertainty and distinguish an
incomplete run from an incomplete inventory. Preserve legacy readers and
default execution behavior. Publish fixtures for these cases; qualification
uses the accepted M44 revision and its frozen dependency-build closure.

Dependencies: Udon M44 accepted and published under separately named authority.
Downstream: Kinet W07.
````

## Status record

````markdown
# Status M90 — Per-step executor outcomes in run evidence

**State:** Completed; implementation accepted and published, review passed in
2 iterations. One execution owner; COMMIT_POLICY: task. Publication authority
and exact source/remote verification are recorded below.

**Scope boundary:** Carry Udon's per-step outcomes into `openudon run` evidence
without absorbing runtime semantics or weakening approval gates. Do not adopt
UWS 1.12, add simulation, contact providers or target accounts, run a public
canary, adopt a runtime, or deploy. Publish the reviewed OpenUdon source only
after qualification and bounded review pass.

**Dependencies.** Udon M44 accepted and pushed (report v5 schema and
fixtures). Planning baseline: OpenUdon
`0c7c5d33da2ba7b190954b9eb402cc14b5ec1f73`, with a clean worktree.
**Downstream.** Kinet W07 re-pins to the accepted, published M90 revision.

**Approved review intake (2026-09-30).** Stage 4 cross-package planning
review (pasted source); source priority and review baseline: not supplied.
Revalidated at `0c7c5d33da2ba7b190954b9eb402cc14b5ec1f73` with
relevant uncommitted planning files included. The user approved these
amendments on 2026-09-30. Primary owner: OpenUdon M90 for F05 (P1); supplies F07 (P1, primary Kinet W06) and consumes F08 (P2, primary Udon M44). Lineage: M90.1–M90.3.
This is intake, not a closing-review iteration; no review counter starts.

**Current evidence.** `internal/udonreport/report.go`, `internal/udonrunner/runner.go` and `internal/trustedrunner/trustedrunner.go`; current readers accept reports v2–v4 and no per-step inventory. Focused owner tests passed.

**Reconciled contract.** Choose the run-evidence version and explicit report-v5 selection contract
before consumer integration. Validate exact run/workflow identity, complete
supported inventory, operation/invocation identities and timestamp/outcome
consistency, with bounded records. Missing, malformed, mismatched or incomplete
inventory never implies not_started; preserve uncertainty and distinguish an
incomplete run from an incomplete inventory. Preserve legacy readers and
default execution behavior. Publish fixtures for these cases; qualification
uses the accepted M44 revision and its frozen dependency-build closure.

Markers: `[ ]` pending, `[~]` in progress, `[+]` complete, `[!]` blocked,
`[-]` closed history, `[X]` cancelled.

## Tasks

| Item | State | Notes |
| --- | --- | --- |
| M90.1 Choose the evidence contract and publish fixtures | `[+]` | Choose evidence version and explicit v5 selection before integration; publish dry-run, success, failed-read/write, interrupted, missing/stale/mismatched report and incomplete-inventory fixtures; define exact run/workflow and step/invocation identity. |
| M90.2 Validate report v5 and record per-step outcomes | `[+]` | Validate selected v5 reports alongside unchanged legacy consumers; bounded identities/counts, inventory completeness, times and outcomes; missing/malformed/mismatched evidence cannot imply not_started; preserve incomplete runs and conservative unknowns without secrets or payloads. |
| M90.3 Qualify against Udon M44, review, and publish | `[+]` | Opt-in qualification at accepted M44 source/frozen build closure with loopback success, failures and kill/checkpoint cases; make check and bounded review with no P1/P2; publish accepted revision only under separately named origin/main authority. |

## Acceptance

Existing v2–v4 report consumers and dry-run evidence are unchanged unless v5
is explicitly selected. Report v5 records are validated against the staged
plan, and malformed, unknown, duplicate, or oversized step records are
rejected. Run evidence carries per-step outcomes with no credential values,
bodies, or headers. The conformance fixtures are published, the opt-in
real-executor qualification passes, and the accepted revision is pushed for
Kinet W07 to pin.

**Udon M44 reconciliation (2026-09-30).** Accepted and published implementation
source `1a5e9aa2045e3d875da2e18aab2d6db869ac5223`; normal push and independent
remote-head verification succeeded. Contract: `udon.execution-report.v5`,
explicit `--execution-report-version v5 --execution-run-id ID`, new report
path; one flat HTTP sequence, 1–256 unique step/operation invocations, complete
ordered inventory and exact workflow-file SHA-256. Reject unsupported shapes
before dispatch. Missing/invalid evidence proves no unstarted step.
Fixtures/schema: Udon `docs/fixtures/execution-report-v5/` and
`docs/schemas/udon.execution-report.v5.schema.json` at that source. Frozen
qualification: `/tmp/udon-m44-qualified-20260930-06/closure.json`, SHA-256
`603a1c4dae5606dd24683ca40050c918e0b7e2b6f31c10e6d4399ce3c5d739c6`;
executor SHA-256 `d2d593ac6d5a6a19406180eb70c5993ee4811fecb20c1a31cdb5a1f59278575b`.
Legacy v3/v4 defaults and v1–v4 schemas remain unchanged. Qualified source
identity is distinct from the subsequent publication/closure metadata commit.

**M90.1 verification (2026-09-30).** Selected `openudon.run-evidence.v3`
for explicit `--executor-report-version v5` only; default v2 and legacy report
selection remain unchanged. `docs/per-step-run-evidence.md` specifies the
exact attempt/workflow/ordered invocation binding, conservative fixed-state
unknown observations and incomplete-run/inventory distinction. Ten observation
fixtures and paired source reports cover dry-run, success, failures, interrupted,
missing/stale/mismatch/incomplete/malformed reports; expected identities and
non-validated unknowns were checked. The frozen report schema is byte-identical
to accepted M44. Implementation/signature/archive wiring follows in M90.2.

**M90.2 verification (2026-09-30).** Strict bounded v5 JSON/semantic validation
now independently checks the frozen public wire contract, exact run/workflow
identity and ordered inventory derived from staged UWS bytes. Opt-in flags
reach internal and canonical external runners; unsupported HTTP sequence
shapes fail before dispatch. V3 retains only validated report references, or
fixed-class all-unknown observations for rejected/missing evidence. Incomplete
runs retain validated durable inventory. V3 signatures, verification and
archives preserve the same binding; raw rejected reports are not archived.
Default v2 and legacy/browser report consumers remain unchanged. Focused full
tests, race tests and vet passed for udonreport, udonrunner, trustedrunner and
CLI (Go 1.26.8; sanitized offline/model-free environment). Conformance, privacy,
time/identity/duplicate/size mutations, external interrupted handoff, signature
and archive checks pass. Real M44 qualification and whole-package gates/review
belong to M90.3. No sibling source or module requirements changed.

## Closing review — iteration 1 started (2026-09-30)

M90 implementation and local qualification are ready; publication remains
separately gated. `make check` and `go vet ./...` passed in the sanitized
Go 1.26.8 offline environment. Logs: `/tmp/openudon-m90-check-01.log` and
`/tmp/openudon-m90-vet-01.log`. Explicit real M44 qualification passes eight
cases at the accepted source/executor/build closure; the Make target's log is
`/tmp/openudon-m90-qualification-01.log`. Focused report/runner/CLI race checks
also pass. Iteration 1 now reviews the complete M90 range from planning
baseline `ab9925ed90e59131d86a4b1a26dc2bd8a6548131`, including current
uncommitted qualification and v46 direction records, against the approved
contract, all compatibility/security/failure boundaries, and Kinet W07.
No publication or acceptance is inferred from task/check completion.

**Review iteration 1 result.** P2 R90-1: an internal runner preparation
failure after inventory derivation can reach the inherited failure-evidence
path with `invoked: true`, although the invocation callback was never entered.
For v5, refuse fabricated invoked evidence and track the actual boundary entry;
preserve legacy behavior. P2 R90-2: the real qualification currently checks
closure field shape and the pinned executor digest but not the exact accepted
closure bytes; bind the recorded closure digest as well. No other P1/P2 in
report identity/inventory/time checks, conservative outcomes, external
canonical handoff, privacy, legacy/browser behavior, signatures or archives.

**Additional iteration 1 finding.** P2 R90-3: encoding/json's case-insensitive
field matching and null-to-zero handling can accept noncanonical v5 property
names or optional null strings despite the frozen closed schema. Enforce exact
field names and non-null fields in v5 only; add alias/null rejection checks.
Legacy decoding remains unchanged. This is still iteration 1, before its
remediation verification and the next review pass.

## Closing review — iteration 2 started (2026-09-30)

R90-1 is fixed with an invocation-boundary observation; v5 refuses to emit
invoked evidence after preparation refusal. R90-2 binds the exact accepted
closure SHA-256 as well as source/executor. R90-3 rejects aliases, nulls and
forbidden optional-field presence, plus timestamp grammar/precision that
cannot be compared faithfully. The legacy reader remains unchanged. Final
`make check`, affected vet and focused race tests pass; the explicit real M44
qualification also passes under race checks after these fixes. Logs are
`/tmp/openudon-m90-check-final.log`, `/tmp/openudon-m90-vet-final.log`, and
`/tmp/openudon-m90-qualification-03.log`. Iteration 2 now reviews the whole
M90 diff and all carried findings. Publication remains separately gated.

**Review iteration 2 result.** Passed; no open P1/P2 or carried findings.
The whole milestone, conformance records, v3 signature/archive verification,
canonical external handoff, pre-dispatch inventory, report privacy, actual
invocation observation, strict JSON/presence/time validation and opt-in frozen
build qualification were reviewed. R90-1/R90-2/R90-3 are closed by their fixes
and passing refusal/canonical-input/real-M44 checks. `make check`, affected vet,
formatting and diff checks pass; focused and real-qualification race checks
pass after the final fixes. Default v2 evidence and legacy/browser report
selection remain unchanged. Current truth and reusable lessons are consolidated;
v46 records the material additive public/private handoff direction.

**Publication gate.** M90.3 is blocked only on separately named normal
fast-forward publication authority for OpenUdon origin/main
(`git@github.com-tabilet:OpenUdon/openudon.git`). The goal's
EXTERNAL_MUTATIONS: none was amended only for Udon M44 publication. It does
not authorize OpenUdon publication. No M90 milestone acceptance, retirement,
Kinet W07 pin or downstream execution is claimed. Final task/source commit
and its clean-source qualification are recorded below before requesting
authority; after publication, reconcile W07 to that exact implementation
revision before closing/retiring M90 and advancing to Kinet M18.

**Clean source qualification (2026-09-30).** Final reviewed implementation
source `ed5b206a524e6e193123b2d06714b75160379560` is committed with a clean
worktree. `make report-v5-qualification` passed again at that exact clean
source; log `/tmp/openudon-m90-qualified-ed5b206.log`. The eight service counts
and exact-attempt observations agree, using the accepted M44 source/binary/
closure digests above. The doc-memory guard also passes; go.mod/go.sum remain
unchanged. Independent `git ls-remote origin refs/heads/main` returned
`0c7c5d33da2ba7b190954b9eb402cc14b5ec1f73`; the local range contains only
the approved M90 plan, prerequisite reconciliation and M90 task commits.
This subsequent readiness record changes no implementation bytes. M90 remains
unaccepted/unretired pending the separately authorized publication.

**Publication authority (2026-09-30).** The user explicitly authorized
OpenUdon M90 publication to origin/main. This selects M90.3 for normal
fast-forward publication of the completed reviewed implementation and its
closure records to `git@github.com-tabilet:OpenUdon/openudon.git`. No other
repository or action is authorized. Remote main was independently verified
at `0c7c5d33da2ba7b190954b9eb402cc14b5ec1f73` before this operation.

**Acceptance and downstream reconciliation (2026-09-30).** Normal fast-forward
publication succeeded and independent remote-head verification returned
`a388b235edc733fd23962ff006d2406d4965b48f`. The accepted implementation is
`ed5b206a524e6e193123b2d06714b75160379560`; readiness/closure metadata does not
change that qualified source identity. Kinet W07 specification and pending
status were reconciled to this exact published implementation, explicit v5/v3
contract, inventory/unknown semantics, admission constraints, fixture locations
and frozen M44 source/build closure in Kinet
`db35053f52fb7ee0e81d833752829b83f76ddd96`. No Kinet implementation or retry
is claimed. All tasks, checks, two-iteration review, consolidation and source
publication requirements are satisfied. Retirement preserves this complete
status/specification; its metadata publication continues under the same
separately named M90 origin/main authority.
````
