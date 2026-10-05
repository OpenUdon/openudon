# Status M97 — Brokered execution handoff

**State:** Execution started, 2026-10-05. M97.1–M97.3 completed; M97.4 is in progress.
**Stage:** STG-09 (Kinet coordination label; milestone IDs remain repository-local).
**Specification:** [M97](milestone.md#m97--brokered-execution-handoff).
**Provenance:** User approved the complete Stage 9 proposal with “Implement the plan” on 2026-10-05. This applies planning-file actions only; a later goal request starts code work. Planning baseline `fbda7e9231b8b306fd1ae3ac623e9d70331b3e08`; [Stage 9 contract](../../../kinet/docs/stage9.md) records discovery evidence and all decisions.

## Scope and dependencies

Own reviewed package/approval/configuration/evidence binding and external private-Udon invocation. Add explicit broker-enabled executor configuration and evidence versions, keeping legacy readers/outputs. No private executor module imports. Bind grant-derived per-run approval to package, concrete inputs/constraints, allowed operations/destinations, executor and credential revisions. Allow the production tier only through this explicit approved broker path; never broaden sandbox destination rules. Values remain outside worker environment, artifacts and reports.

**Requires:** Accepted/published Udon M46 source, contract fixtures and exact executor closure are now reconciled below. OpenUdon baseline fbda7e9231b8b306fd1ae3ac623e9d70331b3e08; existing runtime acceptance c2f161d762bc9f2217bbf0c34b00cdef64b0f7d0 remains frozen.

**Accepted Udon handoff:** M46 source `95c5850fd446e06ac6f79d943774db67e417c989`; independently verified source-record publication `58f9fa5cda92d508e9fcb4085157a09949ddfab9` (completed owner record published and independently verified at `71071537890599e98541abe8ca564490660dfd16`); closing review 2 passed. Frozen executor SHA-256 `53bb9e8976f67c6a5880f07248a99cd195e6285ce7a5a68239793dd5dc6eb429`, fourteen-source closure SHA-256 `a286f21a5dcd4180b7ec19bf0630188ddacec26e0532198db14decd0c86b2069`, broker fixture manifest SHA-256 `7289085b14f744504019ae6d607d351f74522ed3f930ecbbfd159200fd2ddb5f`. Private qualification `/var/tmp/udon-m46-broker-qualification-20261005/closure.json`; use its clean exports and exact executor, not a sibling checkout rebuild. The opt-in contract is `udon.http-broker.v1` plus `--http-broker-config`/report v5, bounded unique straight-line OpenAPI HTTP operations with fixed API-key/bearer bindings; no browser, signing, OAuth, data files, repeated/nested plan or direct fallback. Legacy reports/pins remain unchanged. This satisfies the M46 prerequisite only; M97 implementation and acceptance remain pending.

**Consumers:** Kinet M35 uses the exact accepted/published broker handoff and M97 fixtures; W14 binds approval/evidence and M37 qualifies the bundle. Existing authoring/capture consumers retain their own pins; no W8M adoption.

## Tasks

| Item | State | Notes |
|---|---|---|
| M97.1 — Define approval, configuration and evidence contracts | `[+]` | Publish the broker-enabled versioned handoff with run/grant/policy/credential-reference and exact package/input/executor bindings; preserve existing schemas and readers. Distinguish a bounded recurring grant from the concrete per-occurrence approval emitted by its trusted host. Include mismatch and downgrade refusal fixtures. Defined concrete Authority v1, approval v2, executor config v3 and evidence v4 with strict metadata/digest/deadline/inventory validation, four embedded schemas and seven hashed authority fixtures. Positive Go envelopes validate against the schemas; stale input/credential policy and legacy downgrade tests refuse. Full make fast and focused semantic/schema tests passed offline. Runtime wiring remains M97.2; no new profile is qualified or accepted. |
| M97.2 — Pass broker authority without credential values | `[+]` | Wire the private Unix-socket/capability references through trustedrunner and the external Udon CLI. Preserve production approval checks and sandbox protection; broker mode bypasses environment-value requirements only for declared broker-resolved references. No host sockets, keys or private Udon imports. Implemented explicit private reference through both CLIs and canonical external revalidation; exact authority/compiled input/operation/security/executor preflights; pinned executor/private transport snapshots; credential/proxy environment exclusion; immutable broker config/evidence and durable no-replay claim; evidence v4 verification/sign/archive support. broker-inspect returns exact APItools-backed review metadata; approval-template optionally validates concrete broker authority. Full make fast, focused broker tests and focused races passed offline. Existing schemas/pins/default serialization remain unchanged; actual M46 producer/runtime qualification remains M97.3–M97.4. |
| M97.3 — Qualify producer and consumer fixtures | `[+]` | Consume actual published M46 fixtures and closure, publish a Kinet-compatible positive/negative corpus and manifest, and test replay, stale package/input, unsupported version, wrong operation and uncertainty handling. Keep original producer provenance separate from runtime adoption. Original ten M46 fixtures/manifest copied byte-identically; eleven normalized OpenUdon envelope/inspection fixtures have independent hashes and strict membership/schema/semantic checks. Exact accepted M46 executor passed seven private-socket journeys (success, lost/wrong/unknown/refused/failed/cancelled outcomes), with durable start-before-dispatch, redaction and no-replay assertions. Full make fast, focused corpus and broker races passed offline. Clean-source/native qualification remains M97.4; synthetic corpus provenance is distinct from actual runtime proof. |
| M97.4 — Qualify, review and publish exact handoff | `[~]` | Run owner runtime/adoption checks against exact M46 and preserve historical browser evidence. Persist a whole-diff pre-publication review; this row owns scoped publication only under separately confirmed normal-push authority. Verify actual origin/main and hand off full accepted/source/publication hashes to Kinet before normal closing review/retirement. |

## Acceptance and verification

Approved package and exact run authority are inseparable from broker configuration and evidence. A symbolic credential binding works without an environment secret in broker mode; legacy environment behavior remains unchanged. Unsupported/missing/mismatched authority is rejected before Udon invocation. Accepted source, fixtures and publication are consumable by Kinet.

make fast for routine edits; go test ./...; go vet ./...; make check; go run ./cmd/openudon check; go run ./cmd/openudon check-apitools-boundary; (cd tabilet && go run ../cmd/openudon check-doc-memory); fixture validation and git diff --check. Run the required affected make smoke and full make qualify for runtime adoption, including three fresh native repeats under owner rules, in disposable exact-source closures. No cache result substitutes for required fresh qualification.

## Execution and authority

One execution owner works serially in the approved cross-package order:

```text
Kinet:M34 -> Udon:M46 -> OpenUdon:M97 -> Kinet:M35 -> Kinet:A14 -> Kinet:W14 -> Kinet:M36 -> Kinet:U12 -> Kinet:M37
```

The later confirmed goal uses Kinet's existing tabilet/GOAL.md as coordinator with COMMIT_POLICY: task; each package's instructions, local ledger, verification and closure remain authoritative. Kinet holds the only launch reference. Original planning approval wrote plans only. The later Stage 9 goal confirmed task commits and scoped fetch/normal publication of reviewed M97 implementation and closure records to git@github.com-tabilet:OpenUdon/openudon.git. No downloads, installs, deployment, live API/model calls or real email is authorized. This milestone's final row includes a separately authorized publication operation: it must be in progress before its launcher, source must pass pre-publication review, and task state never substitutes for authority. Finish the ordinary post-task closing review after publication; carry its persisted counter across interruptions.

Reconcile actual full accepted/source/publication revisions and hashes before starting consumers. Future upstream hashes are **not yet available**; never invent them or substitute current sibling worktrees for frozen qualified inputs. Preserve unrelated changes and frozen evidence. After verification, persisted review and downstream reconciliation pass, perform this repository's normal retirement. Do not reopen historical milestones.

## Persisted review

- Review iteration: **1/10**; pre-publication whole-milestone review started and passed on 2026-10-05 after the corrected candidate passed every required automatic gate.
- Findings: pre-publication review 1 passed with no open P1/P2-or-higher finding. The full baseline-to-corrected-source diff was checked for authority/version correspondence, legacy bytes, private APItools/executor boundaries, pinned transport/executor snapshots, no replay, conservative report-v5 outcomes, both evidence copies and exact fixture/schema/build provenance. Publication and ordinary closing review remain required.
- Reviewed qualified implementation revision: `f4127c159e18fa66619659bc3c4b8757b7022267`; final acceptance awaits publication and closing review.
- Verification/build evidence: corrected clean source `f4127c159e18fa66619659bc3c4b8757b7022267` passed make check, go vet ./..., all seven exact-M46 runtime cases, fresh registration smoke and all 39 native stages with independent report verification and joined private-display teardown. Full make fast and focused broker/authority races also passed. Exact artifact and evidence bindings are in docs/m97-qualification.md; acceptance still requires review and publication.
- Publication evidence: pending. The separately confirmed Stage 9 request authorizes scoped normal origin/main publication after this pre-publication gate; planning alone grants none.
- Downstream reconciliation: pending exact upstream/downstream revisions.

## M97 temporary display authorization — 2026-10-05

The user explicitly authorized the installed Xvfb on vps-f7dfc687 for M97's
required smoke and three fresh native qualification passes. TCP stays disabled,
X authentication is private/temporary, Chromium sandboxing remains enabled and
all owned processes are torn down. Only disposable local fixtures are included;
no installation, public-service access or other milestone display is authorized.
The exact M97.4 operational row must be in progress before its launcher.

## M97.4 qualification selection — 2026-10-05

Operational row entered in progress before any display/browser launcher.
Candidate application/test source: `c55fb8eb208802955c8aac3e8ec6dc61b2d80bfc`.
Use a clean detached local export with nineteen source entries: exact current
Browserdriver/Browsertools/UWS locks and the fourteen-source historical M45
browser closure. Broker checks independently retain the exact published M46
executor/closure; historical browser pins are not substituted with M46.
Installed Browserdriver modules match all four package-lock versions; Node
24.14.1, npm 11.18.0, cached Go 1.26.6, Playwright 1.62.1 and Chromium
151.0.7922.34 are available. The installed root-owned setuid Chrome helper at
`/opt/google/chrome/chrome-sandbox` supplies sandboxing; no sandbox-disable
override is allowed. Native input inventory will bind the actual tool/driver
bytes before launch. Temporary Xvfb authority is recorded above.

Estimated owner run: 30–50 minutes including offline gates, affected fresh
registration smoke, all 39 native stages and independent report verifiers.
Source cloning/build outputs are disposable; there are 31 GiB free on the
existing disk. No asset download/install, public target or provider call is
authorized. This selection is a preflight record, not passing evidence.

The first browser-free input attempt refused the inherited `0002` umask.
A disposable-process `0022` selection passed the unchanged input validator
(v3 identity `2b6794cd92c90d89b678270f40fad55fe6e4c21d1662b54368fa41e8140dcbb5`).
The display-bound input will be recorded separately because private X authority
and the closed launch environment are additional inputs. No browser was started
by these checks and no global umask/security policy was changed.

Fresh registration smoke passed in 74.059 seconds (development evidence only).
Full qualify attempt 1 stopped at offline `openudon_unit`: link output hit the
existing `/tmp` tmpfs user quota, despite 30 GiB free on `/var/tmp`'s disk.
All owned display processes joined, socket disappeared and private X authority
was removed. Failed report/diagnostic/launch records remain private under
`/var/tmp/kinet-stage9-p_9plhx7/m97-native/`; no native acceptance is claimed.

Retry selection retains the same clean source and installed assets. A private,
byte-bound Go launcher calls the exact cached Go 1.26.6 binary while redirecting
its temporary files to the disposable disk root and limiting build concurrency
to two. The qualification's closed child environment otherwise drops these
resource settings. This changes no validator, stage count, sandbox policy or
source bytes. No shared `/tmp` data or global Go configuration is removed.
Retry runs all required offline gates and three fresh native repeats; the
earlier successful smoke retains its original execution/input context.

Attempt 2 also failed at the initial offline gate: Go's `go run` prepends
`GOROOT/bin` to its child PATH, selecting the real Go binary ahead of the
resource launcher. That downstream gate consequently still used quota-limited
`/tmp`. Failure reports and verified display teardown remain in `attempt-2/`.
The next attempt uses the clean, `vcs.modified=false` candidate CLI directly
through Make's GO launcher, preserving the owner's documented direct-CLI
qualification entry point and all exact-source gates. The Go launcher remains
on child PATH for disk-backed build resources; its bytes and the actual CLI
are bound in the fresh native input identity. No test, count or policy is changed.

Attempt 3 passed direct-binary input preflight but the disposable Python launcher
then stopped on an undefined launcher-path variable before any qualification
stage. Its owned display/authority teardown passed. That local launcher typo
is corrected; attempt 4 keeps fresh separate outputs. These preparatory failures
are not review iterations or native execution evidence.


Attempt 4 passed all four offline gates and ten stages of native pass 1, then
failed registration_capture_handoff with the fixed registration_execution code.
It is failed evidence, not a partial qualification. Display teardown passed.
Focused diagnostic-only BRP checks subsequently passed with disk-backed and
default temporary roots; the exact candidate component also passed, and a
private diagnostic-only logger overlay passed under the same closed environment.
No runtime/schema/fixture bytes changed; diagnostic overlays are excluded from
qualification. The original failure was not reproduced and its cause is not
claimed. Attempt 5 reruns the entire unchanged qualification with separate fresh
outputs; it consumes no development or failed-run cache.


M97.4 task inspection confirmed an async-sidecar argv copy retained the private
broker config path although the main run record redacted it. The added portable-
evidence assertion failed before the fix. Both argv copies now use the existing
broker redactor; legacy evidence is unchanged. A clean source checkpoint is
required before the full qualification restarts. Attempt 5 was deliberately
interrupted before native stages; its prior gates and joined display teardown
remain private, and none is acceptance. This is task verification, not a
completed milestone review iteration. Broker production approval and API-key
header/query metadata coverage are also included in this source checkpoint.


Corrected-source checkpoint verification passed: full make fast, focused broker
corpus/production/API-key tests and broker/authority races. git diff --check is
clean. The doc-memory evolution warning was inspected: this is an implementation
advance within approved M97, so v49 remains current. Native qualification and
both publication/closing reviews remain pending; M97.4 stays in progress.


## Corrected M97.4 native selection — 2026-10-05

Clean candidate source: `f4127c159e18fa66619659bc3c4b8757b7022267`; no worktree overlays.
Private frozen root: `/var/tmp/kinet-stage9-p_9plhx7/m97-final`. Source archive SHA-256
`87b14b27bf5c1b72280cdaab3293ec8207809e10780eace66885070c0cbcb5ec`. CLI digests:
openudon `4f0c6cad518331f4dcf61de986c4b21fbda9aadaf1065a2efaafbda2f1220c17`;
udon-runner `5e0cf3623db21a2aef72e7f5daefa34ae8672306658c35222995e4af3735407e`.
Both build records report vcs.modified=false at that exact revision.
Build-input inventory SHA-256
`38a2b26a19ab073fddb497331bd6877b43756e18ddbb3ef110c73d923436a324` binds
48 modules, 2,784 compiled source/embed files and all seventeen clean exported
repositories (nineteen native report source entries). Fixture/schema hashes and
M46 executor/closure remain unchanged. The private launcher is byte-bound and
uses the exact compiled CLI, cached Go and disk-backed Go temporary files.

M97.4 remains in progress before its fresh affected smoke and full 39-stage
qualification. Installed assets, prior Xvfb authority, private authentication,
sandbox enabled and joined teardown remain the selection; estimate 30–50 minutes.
The corrected source does not consume old smoke/native results or logger overlays.
No acceptance/publication is claimed by this selection.


The corrected candidate's first launcher passed input preflight but stopped on
a private-script variable typo before smoke/native work; display teardown passed.
The corrected launcher uses explicit tool paths and separate qualification-2
outputs. This is not native failure or review evidence; source stays f4127c1.


## M97.4 exact qualification and pre-publication review

Corrected source passed all four offline gates, all 39 fresh native stages and
independent verification; fresh smoke passed with reuse=false. Joined display
teardown passed all three checks. Exact-source make check, go vet and all seven
actual-M46 private-broker cases also passed. Qualified closure SHA-256
`5fb02718562932acd64c2e1a19245e0773441a0e718ddba577aa90b15694035c`; full artifact/evidence hashes are in
[the handoff](../../docs/m97-qualification.md). Earlier contexts remain excluded.
Review 1 inspected the whole milestone, not only the checkpoint patch; no open
P1/P2 finding remains. M97.4 stays in progress through its authorized normal
publication. Ordinary post-task closing review and downstream reconciliation
remain pending; this record does not claim them complete.
