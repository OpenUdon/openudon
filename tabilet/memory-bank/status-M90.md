# Status M90 — Per-step executor outcomes in run evidence

**State:** M90.1–M90.2 complete; M90.3 pending; Stage 4 goal execution approved 2026-09-30.
One execution owner; COMMIT_POLICY: task. OpenUdon publication is not authorized.

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
| M90.3 Qualify against Udon M44, review, and publish | `[ ]` | Opt-in qualification at accepted M44 source/frozen build closure with loopback success, failures and kill/checkpoint cases; make check and bounded review with no P1/P2; publish accepted revision only under separately named origin/main authority. |

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
