# Status M96 — Reviewed capture package authoring

**State:** M96.1–M96.3 verified, 2026-10-01; M96.4 in progress. Native qualification and milestone acceptance remain unproved.

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
| M96.1 — Freeze reviewed capture package contract | `[+]` | Versioned bounded request/result and approval bindings, public fixtures for authenticated/TOTP and registration, provenance and compatibility. |
| M96.2 — Expose neutral adoption and package authoring | `[+]` | Native receipt/source validation and exact approval reuse neutral engine/materialization/writer; real main CLI dispatch; no iCoT or capture replay. |
| M96.3 — Verify conformance and refusal behavior | `[+]` | Owner/revision/digest/mode/policy/path/replay/interruption and side-write checks; retained legacy/protected fixtures unchanged. |
| M96.4 — Qualify, review and publish replacement | `[~]` | Frozen exact-source synthetic loopback capture→adoption→build→prepare/promote/inspect/recovery for both modes/TOTP; three fresh native repeats and required integration gates; bounded review, publication and downstream reconciliation before retirement. |

## Acceptance and verification

Run make check, formatting/vet, affected race and real-dispatch conformance; inventory protected fixture bytes. Run required integration/current-native qualification once on the frozen candidate, including three fresh native repeats under AGENTS.md; retain actual source/binary/build closure and failures. A capture receipt alone is not a reviewed/built package. Exercise separate approvals and inspect/recovery after uncertainty without replay. No live account, registration submission, runtime operation or model disclosure. Native registration recipe capture remains supported.

Temporary already installed Xvfb on `vps-f7dfc687.vps.ovh.us` is authorized for M96.4 synthetic loopback qualification only: private temporary X authentication, TCP disabled and automatic independently verified teardown. No install, permanent service or public listener. The expired M93 desktop is never restarted.

Normal bounded review: persisted count 0/10, not started. Publish scoped source/qualification/closure to existing origin/main with the confirmed goal's prepush exact-diff recording and independent remote verification. Reconcile Kinet M19/U07/M20 and pending M95 to the actual published accepted revision before consumers advance; no guessed future hash. Kinet's launcher remains the only coordination reference.

## Provenance

Kinet M19.3 execution discovery at `c2243c93fe3fce08847eaf425722e30a75c3cf0d` against exact published producer `04dacce77a29f5e6db427dc47e3ed9ef766b2c32`. `authoringui/application_transaction.go` exposes capture adoption only through retained iCoT; neutral engine/writer already exist. Package preparation requires ordinary review/build evidence. Disposable diagnostic `/var/tmp/kinet-m19-native-contract-pjc2enyk/contract-observation.json` copied unchanged public M93 artifacts and deliberately unchanged example intent: it has additional expected operation/assembly failures and is not accepted qualification. User approved the complete proposal, exact file actions and subsequent goal/publication/display extension on 2026-10-01. User separately requires a pause before W8M W28 to verify W27's latest status; do not assume W27 remains active or is complete.

## Execution selection — 2026-10-01

One owner resumes the user-approved extended task-policy goal at planning commit `923ec73`. M96.1 is the sole general in-progress row; Kinet M19.3 remains blocked. The mandatory pre-W28 pause is preserved. Reuse neutral engine/writer; no UI/iCoT call, native capture rerun or runtime authority.

## M96.1 verified task

Closed bounded browser-author v1 request/plan/result types and six synthetic public fixtures freeze separate read-only catalog/planning and exact-confirmed native apply. Strict native start, both modes/TOTP, symbolic-only bindings, receipt paths and exact plan identity/partial-write semantics tested. Full make check, affected wire race and diff checks passed. Frozen baseline/hashes/logs: `/var/tmp/openudon-m96-1-qualified-llwa83ui`; 53 protected tracked fixture/schema files unchanged. No command dispatch, browser, authoring write, promotion or runtime ran. M96.2–M96.4 and review0/10 remain pending.

## M96.2 selection

M96.1 committed at `5dfd12625b309dc08c3620c65d0b7d72b1731f3d`. M96.2 is the sole general in-progress row. Native receipt/source reconstruction and lowering stay OpenUdon-owned; Kinet M19.3 remains blocked until milestone acceptance/publication.

### M96.2 verified implementation

Unpublished v1 start transport corrected to base64 exact native bytes, preserving receipt hashes under outer JSON formatting; original M96.1 wording retained in the knowledge journal. Native file actions are also bound into the plan. Reuse pure elicitor lowering plus artifactwriter directly: interactive Engine.Open would rediscover already imported physical profiles and reject their virtual-source collision. Its safety policy is unchanged; no new writer/engine or UI import. Exact full-inventory precommit guard uses only the existing writer's explicitly reported transient paths; broad suffix exclusions are forbidden.

Initial tests found omitted required workflow output and a guard seeing the native writer's own staged files. Native result output lowering and an additive observation callback on the one existing writer corrected both; legacy callbacks retain their behavior. Both synthetic authentication/TOTP and registration plan→apply→build and replay refusal now pass. These are offline native-library checks, not real browser qualification or milestone acceptance.

Full make check, affected race (browserpackage, artifactwriter, browserauthoring, browsercandidate, elicitor, main CLI), full Go vet and affected formatting passed. Frozen candidate bytes/logs: `/var/tmp/openudon-m96-2-qualified-abrk281_`, based on full `5dfd12625b309dc08c3620c65d0b7d72b1731f3d` plus recorded task bytes. All 53 protected pre-M96 fixture/schema bytes remain unchanged. No browser, model, promotion or executor ran. M96.3/M96.4 and review0/10 remain pending.

## M96.3 selection

M96.2 committed at `7eb3c9634a626fb81b0f88369ee656eeab8faeb7`. M96.3 is the sole general in-progress row; Kinet M19.3 remains blocked. Test exact-key wire closure, side-write/temporary guards, stale native evidence, cancellation/output loss and legacy preservation before native qualification.

### M96.3 conformance progress

Focused checks exposed encoding/json case-insensitive aliases and missing generated-artifact byte bindings; closed exact keys and native prepared-file digests now address both. Native cleanup warnings are retained as a bounded `cleanup_required` flag, without paths/text disclosure. Inventory validates directory ownership/modes too. Read-only catalog/unbound/conflict refusals, mode/path/native-policy drift, expiry, private writer transient guards, cancellation and output loss/replay are tested. Initial conflict-readiness assertion incorrectly ignored explicit allow_overwrite; corrected before rerunning checks. No native/browser acceptance is claimed.

### M96.3 verified task

Full make check, affected race, full Go vet, formatting and diff checks passed on baseline `7eb3c9634a626fb81b0f88369ee656eeab8faeb7` plus frozen task bytes in `/var/tmp/openudon-m96-3-qualified-5l48zy67`. All 53 protected pre-M96 fixture/schema files remain unchanged. The real dispatch and library refusal/interruption tests are offline; M96.4 browser qualification/review/publication remain required.

## M96.4 selection

M96.3 committed at `8c2606451a4d012d7418d3b539a87a27dcb50275`. M96.4 is the sole general in-progress row. Freeze that exact application source and locked dependencies, run fresh real main capture→author→build→native package lifecycle plus native3/integration, then persisted review/publication/reconciliation/closure. The authorized private temporary Xvfb is new; M93 historical evidence/desktop remain untouched.

### M96.4 qualification discovery — origin policy

Exact frozen candidate `8c2606451a4d012d7418d3b539a87a27dcb50275` passed fresh main capture→author/build→prepare/promote/inspect/recovery for authenticated/TOTP, retained registration v2 and typed registration v4. During independent policy diagnostics, a resealed receipt/start with an added origin still produced a ready plan (`/var/tmp/openudon-m96-policy-diagnostic-3_vfos9n/observation.json`). No writes ran in that diagnostic. Native adoption now compares immutable start/review origins against transaction provenance; fresh refusal tests cover the substitution. The first frozen candidate is not accepted. Corrected exact-source qualification is required; its old runs retain actual identities and cannot be upgraded into acceptance. This is task qualification discovery, not the closing review (still0/10).

Origin-policy correction passed fresh full make check, affected browserpackage/browserauthoring race, full vet and diff checks; exact changed bytes and check-log hash are recorded in the first frozen bundle `origin-correction-context.json`. The original concurrent integration attempt recorded12 pass/5 failure/3 optional unrequested; failure diagnostics remain private. A direct Chromium inventory rerun passed. Its failed report is retained, not treated as acceptance; corrected candidate receives fresh required gates.

The integration report's first failure followed the source-state recheck at the isolated Udon gate; remaining unrequested mandatory gates were conservatively marked failed, rather than all five executing and failing. Independent supplied source trees are clean afterward, and Chromium inventory succeeds. Native and integration gates were launched concurrently against the same supplied snapshot, which can expose temporary test activity to source checks. The exact cause is not yet established; fresh integration will run only after native gates finish, preserving source checks and retaining the actual failed attempt. No source-check bypass or increased timeout is authorized or used.

A final wire inspection found encoding/json also accepts numeric arrays into []byte. The frozen public start transport is a base64 string, so numeric-array starts are now explicitly refused and covered by conformance. This correction changes the candidate source: prior native attempts are superseded qualification, never accepted as the final source. No wire version or external Kinet envelope changes.

Superseded native attempts were stopped by SIGTERM to their exact owned OpenUdon evaluation processes, allowing native context cancellation/child joins. Both supervisors independently verified owned Xvfb exited and temporary X authority directories were removed; neither attempt is acceptance. Fresh exact start-transport refusal race passed. A final corrected frozen candidate and serial qualification are next.

Canonical-start compatibility was also checked against the existing native configuration: captured role/context/label/profile identifiers and origins use native textual/URL normalization, while receipt approval still hashes exact original start bytes. Imported validation now reuses native origin canonicalization and preserves those equivalent forms without permitting an added origin. The first positive fixture failed because Go's nested test directory was not private; explicit0700 fixed that fixture, without weakening production checks. Native conformance and final qualification remain pending.

Final normalization correction passed full make check, affected race, full vet and diff checks on baseline `6d429f8a61d0a071527447a5035f79ae95bf2255` plus frozen exact bytes in `/var/tmp/openudon-m96-final-corrections-rntjsypu`. The positive fixture now checks BOTH native start schema and native normalized configuration; its first private-directory and invalid whitespace-profile-ID setup failures remain in earlier logs. Immutable original start bytes remain receipt-bound, while equivalent native role/context/label/origin representations are accepted.

Final candidate `e091c7c689e75b74dca746faa26b2a3172835c0d` is frozen at `/var/tmp/openudon-m96-qualified-bbt7i2p0`: build and full offline checks passed; all three fresh public capture→author/package/recovery journeys passed. Native3 failed before stages with `current_stack_source_state`. Read-only inspection identified an ignored empty `.openudon-run/` left by default checks in the owned clone; all other supplied trees were clean. Recorded and removed exactly that owned empty directory, retaining code/source checks unchanged. Failed invocation/display/log context is preserved in `native-attempt1-preflight`; teardown was verified. Retry only the affected native gate on clean exact source, retaining the actual already-passed capture evidence; integration remains serial and pending.

Further read-only immutable-start diagnostics at exact candidate `e091c7c689e75b74dca746faa26b2a3172835c0d` found resealed starts with changed authentication login/dashboard and registration URL still produce ready plans. Registration profile-ID substitution also remains accepted; its evidence representation must be reconciled with native import semantics before claiming a refusal. Diagnostic `/var/tmp/openudon-m96-start-diagnostic-0_m2vc8z/observation.log` changed only owned synthetic evidence and performed no authoring writes. Final candidate is not accepted. This task qualification correction remains before the closing review (0/10); retain current attempt as superseded and verify cancellation/teardown before freezing corrected source.

Authentication login/dashboard comparisons now reuse native URL normalization and the native recipe/success proof; resealed substitutions are refused. Both modes also explicitly test changed starts against the original receipt. Registration native semantics permit navigation among approved origins and do not retain a separate capture profile ID/initial URL in the recipe: no invented equality or navigation restriction was added. Content-addressed receipt/start identities must come from the original approved capture; unsigned replacement receipts are not new capture attestation. This limitation is documented. Full make check, affected race, full vet and diff passed at exact e091c7c baseline plus hashed correction bytes in `/var/tmp/openudon-m96-login-correction-ndejh2_u`; its initial setup-only missing-sibling check failure is retained separately. All53 protected schema/fixture files remain unchanged. Superseded native attempt joined and private display teardown independently passed. Fresh native qualification, persisted closing review and publication remain pending.
