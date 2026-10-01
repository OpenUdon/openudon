# Status M96 — Reviewed capture package authoring

**State:** Approved planning, 2026-10-01; all rows pending. No implementation or acceptance is inferred.

**Goal.** Make a reviewed native capture usable for ordinary browser source/step/package authoring without iCoT.

**Dependencies.** Accepted and published M91–M94; exact current producer baseline `04dacce77a29f5e6db427dc47e3ed9ef766b2c32`. Existing neutral authoring engine, elicitor, artifact writer and package pipeline remain the implementation owners.

**Downstream.** Kinet M19.3/M19.4, then U07 and W8M W28; M95 retains this replacement during removal; M20/W29 qualify final adoption.

Specification: [milestone.md](milestone.md#m96--reviewed-capture-package-authoring).

Markers: `[ ]` pending, `[~]` in progress, `[+]` complete, `[!]` blocked, `[-]` closed history, `[X]` cancelled. One general row in progress across the coordinated ledgers. Task commits follow the confirmed goal's `COMMIT_POLICY: task`; review and closure follow this package's rules.

## Scope and contract

Expose an additive bounded public non-iCoT command to adopt an exact reviewed native receipt and select its native virtual browser operations, obtain ordinary authoring approval, materialize the reviewed artifacts and build the package through existing neutral implementation. Freeze command spelling and versioned wire in M96.1. Bind current package/input revision, receipt and transaction digests, source/operation identities and immutable mode/profile/origin/goal/dashboard/TOTP/registration constraints. Credentials remain symbolic; never accept values or widen capture authority. Retain separate capture import, authoring and package promotion approvals. No browser rerun, runtime invocation, UI/listener, duplicate semantic writer or automatic promotion is included.

Reject malformed/unknown fields, stale input/receipt/revision, changed mode/policy, unreviewed sources, unsafe/overlapping paths, side writes and replay. Interrupted output grants no replay authority; inspect package evidence before further mutation. Preserve v1 capture, existing commands and legacy package/default/fixture bytes. Use neutral elicitor/authoringengine/artifactwriter, not UI/controller imports. Do not reopen M93 or any retired record.

## Tasks

| Item | State | Notes |
| --- | --- | --- |
| M96.1 — Freeze reviewed capture package contract | `[ ]` | Versioned bounded request/result and approval bindings, public fixtures for authenticated/TOTP and registration, provenance and compatibility. |
| M96.2 — Expose neutral adoption and package authoring | `[ ]` | Native receipt/source validation and exact approval reuse neutral engine/materialization/writer; real main CLI dispatch; no iCoT or capture replay. |
| M96.3 — Verify conformance and refusal behavior | `[ ]` | Owner/revision/digest/mode/policy/path/replay/interruption and side-write checks; retained legacy/protected fixtures unchanged. |
| M96.4 — Qualify, review and publish replacement | `[ ]` | Frozen exact-source synthetic loopback capture→adoption→build→prepare/promote/inspect/recovery for both modes/TOTP; three fresh native repeats and required integration gates; bounded review, publication and downstream reconciliation before retirement. |

## Acceptance and verification

Run make check, formatting/vet, affected race and real-dispatch conformance; inventory protected fixture bytes. Run required integration/current-native qualification once on the frozen candidate, including three fresh native repeats under AGENTS.md; retain actual source/binary/build closure and failures. A capture receipt alone is not a reviewed/built package. Exercise separate approvals and inspect/recovery after uncertainty without replay. No live account, registration submission, runtime operation or model disclosure. Native registration recipe capture remains supported.

Temporary already installed Xvfb on `vps-f7dfc687.vps.ovh.us` is authorized for M96.4 synthetic loopback qualification only: private temporary X authentication, TCP disabled and automatic independently verified teardown. No install, permanent service or public listener. The expired M93 desktop is never restarted.

Normal bounded review: persisted count 0/10, not started. Publish scoped source/qualification/closure to existing origin/main with the confirmed goal's prepush exact-diff recording and independent remote verification. Reconcile Kinet M19/U07/M20 and pending M95 to the actual published accepted revision before consumers advance; no guessed future hash. Kinet's launcher remains the only coordination reference.

## Provenance

Kinet M19.3 execution discovery at `c2243c93fe3fce08847eaf425722e30a75c3cf0d` against exact published producer `04dacce77a29f5e6db427dc47e3ed9ef766b2c32`. `authoringui/application_transaction.go` exposes capture adoption only through retained iCoT; neutral engine/writer already exist. Package preparation requires ordinary review/build evidence. Disposable diagnostic `/var/tmp/kinet-m19-native-contract-pjc2enyk/contract-observation.json` copied unchanged public M93 artifacts and deliberately unchanged example intent: it has additional expected operation/assembly failures and is not accepted qualification. User approved the complete proposal, exact file actions and subsequent goal/publication/display extension on 2026-10-01. User separately requires a pause before W8M W28 to verify W27's latest status; do not assume W27 remains active or is complete.
