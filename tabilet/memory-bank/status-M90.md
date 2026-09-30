# Status M90 — Per-step executor outcomes in run evidence

**State:** Pending; planning approved 2026-09-30 as part of Kinet's stage 4
cross-package plan. No row has started.

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
| M90.1 Choose the evidence contract and publish fixtures | `[ ]` | Choose evidence version and explicit v5 selection before integration; publish dry-run, success, failed-read/write, interrupted, missing/stale/mismatched report and incomplete-inventory fixtures; define exact run/workflow and step/invocation identity. |
| M90.2 Validate report v5 and record per-step outcomes | `[ ]` | Validate selected v5 reports alongside unchanged legacy consumers; bounded identities/counts, inventory completeness, times and outcomes; missing/malformed/mismatched evidence cannot imply not_started; preserve incomplete runs and conservative unknowns without secrets or payloads. |
| M90.3 Qualify against Udon M44, review, and publish | `[ ]` | Opt-in qualification at accepted M44 source/frozen build closure with loopback success, failures and kill/checkpoint cases; make check and bounded review with no P1/P2; publish accepted revision only under separately named origin/main authority. |

## Acceptance

Existing v2–v4 report consumers and dry-run evidence are unchanged unless v5
is explicitly selected. Report v5 records are validated against the staged
plan, and malformed, unknown, duplicate, or oversized step records are
rejected. Run evidence carries per-step outcomes with no credential values,
bodies, or headers. The conformance fixtures are published, the opt-in
real-executor qualification passes, and the accepted revision is pushed for
Kinet W07 to pin.
