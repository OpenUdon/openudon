# Status M91 — iCoT inventory and behavior-preserving extraction

**State:** M91.1–M91.5 complete; M91.6 resumed with explicit temporary test-display authority. Native qualification/review/acceptance remain incomplete.

**Goal.** Move shared implementation out of iCoT while keeping all current consumers working.

**Dependencies.** Accepted/published M90 at ed5b206a524e6e193123b2d06714b75160379560; current P07/P08 and E23/E24 behavior. Udon M45 precedes this milestone in the serial launch order, not as an extraction API prerequisite.

**Downstream.** M92–M95; Kinet W08; M93 supplies the W8M replacement contract.

Markers: `[ ]` pending, `[~]` in progress, `[+]` complete, `[!]` blocked,
`[-]` closed history, `[X]` cancelled. (OpenUdon's convention; W8M's differs.)

## Scope and contract

Inventory terminal authoring, ui/control protocols, browser authoring/transactions, lint/repair/reconcile/report, evaluation/variants/scorecards, corpus/provider tooling, worker dispatch, qualification, docs and W8M consumers. Retain capabilities through replacement commands or Kinet; do not silently discontinue an unmatched journey. Keep registration and authenticated/TOTP capture. Present any additional discontinuation for explicit approval before removal. Extract artifactwriter, review/sanitization, source discovery/catalog planning/session types and browser/qualification helpers into neutral packages, without duplicating them. Rebase non-iCoT importers and evaluation; iCoT remains functional in 5A. Preserve P07/P08 v11 dispatch and E23/E24 current-stack inputs, including any already-landed W27 upstream handoffs.

## Tasks

| Item | State | Notes |
| --- | --- | --- |
| M91.1 — Inventory journeys and consumers | `[+]` | Record retained/replaced/discontinued journeys and replacement checks; inventory both W8M capture modes, expert CLI and evaluation surfaces; approve any new discontinuation explicitly. |
| M91.2 — Extract artifact writing and draft review | `[+]` | Move transactional artifactwriter and shared review/sanitization into neutral implementation; prove byte-equivalent fixtures. |
| M91.3 — Extract discovery and session logic | `[+]` | Move local/catalog discovery, planning and session types; decouple non-iCoT authoring consumers from Authoring icot. |
| M91.4 — Extract browser worker and qualification helpers | `[+]` | Move process dispatch/launch and scenario/registration helpers; preserve both capture modes and current-stack inputs. |
| M91.5 — Rebase evaluation | `[+]` | Move lint/evaluation, variants and scorecard callers off cmd/icot without changing their fixture corpus or expected coverage. |
| M91.6 — Prove equivalence, review and publish | `[~]` | Check imports, fixtures, evaluation and owner qualification; preserve P07/P08 dispatch; publish accepted source. Consumer qualification belongs to Kinet W08, without weakening its production pin. |

## Acceptance and verification

Step conformance fixtures remain byte-identical and evaluation/qualification results comparable. No non-iCoT importer depends on internal/icot or authoring/icot. Qualify extraction within OpenUdon; Kinet's existing script always checks pinned M90 and is not evidence for this revision. Kinet W08 subsequently qualifies adopted M92 containing this extraction. Run make check, owner fast/smoke gates and required frozen integration qualification; bounded review and publication.

Default checks use fake providers, disposable roots and loopback fixtures. No live target operation or deployment is authorized. Preserve package instructions, one execution owner and exact upstream reconciliation before advancing. Task commits/publication follow only the separately launched goal's explicit policy; this planning approval performs neither.

## Provenance and lineage

Source: “Stage 5 draft reconciliation” (Kinet `stage5/REVIEW.md` findings F01–F10 and its accompanying drafts; refinement findings SR01–SR06; see the findings table in Kinet `docs/kinet-order.md` §6). Source priorities: not supplied. Draft baseline: e12a6488b86cafddb9298c7917de84fbc1cc85ff. Revalidated at this repository's full HEAD `e12a6488b86cafddb9298c7917de84fbc1cc85ff`. The draft source is read-only. Relevant uncommitted evidence: Kinet's `stage5/` drafts and APItools' approved M81/M80 planning changes; no implementation changes were used or made. User approved the complete dispositions and planning-file actions on 2026-09-30.

F03 (not supplied / P1, confirmed) is owned here: stepauthoring/{bind,source,flow_review}.go, browserscenario/, eval/authoring_variants.go and authoring/ import iCoT code. F09 (not supplied / P2, confirmed) is owned here: preserve internal/{trustedrunner,udonrunner}/browser.go and internal/browsersystem/inputs.go. M95 consumes that compatibility evidence.

Lineage: M87–M90 remain accepted step/handoff foundations; E23/E24 are retired. Preserve E21/P07/P08 completed rows and review counters; ordinary closure is separate from this planning intake.

## Closing review

Persisted iteration count: 0/10. Not started; this reconciliation is intake, not a closing-review iteration. Resume any interrupted future review at its persisted number. Acceptance, exact source/build revisions, publication and downstream reconciliation remain pending and must be recorded from observed evidence before normal package retirement.

## Execution reconciliation — 2026-09-30

Udon M45 precedes M91 in the approved serial order and is now accepted with
published closure `6c4fb8c80a06179f8c3e33ebe685b2d44f66d273`. Its qualified
source is `238f2e487d50ffec057b7a109a35c9db03f59c55`; M92 owns adoption,
not this extraction. APItools M81/M80 remain accepted/published prerequisites
for M94. The producer handoffs are recorded in each pending consumer status.

E21/P07/P08 terminal task rows and passed review counts were preserved and their
unfinished normal history retirement completed before M91. They resolve through
OpenUdon's history index; no old task, review, browser run or W8M claim was
reopened. P08's v11 pairing is the current behavior; P07's v10 preparation
result retains its historical scope. No W27 capture/live operation was touched.

The user updated the confirmed launch reference to automatically authorize
scoped pushes, including exact-diff approval, after the execution owner's
recorded diff/remote/verification checks. Other named human checkpoints remain.
M91.1 inventory dispositions still require user approval under that reference;
Gate 5B later also requires accepted replacements and demonstrated journeys.

## M91.1 inventory prepared

[Retained-journey inventory](../../docs/icot-retirement-inventory.md) contains
23 entries covering terminal/seed/replay flows, source discovery, transaction
writing, step commands, both browser captures and TOTP, UI/control, workers,
expert review/repair and evaluation, legacy packages, v11 dispatch,
qualification, W8M authority and release gates. Every entry has an existing
package owner and explicit replacement evidence; all remain available in 5A.
There are zero proposed additional capability discontinuations.

Current import inspection confirms the extraction seams, including Authoring's
neutral readiness/session primitives and transport-specific progressive
adapters. The inventory is reviewable; no source file, provider/account,
workflow artifact or qualification binding changed. M91.1 stays in progress
at the retained inventory-disposition human checkpoint before extraction.
The later Gate 5B still requires completed replacements and both demonstrations.

## M91.1 approved and complete

The user explicitly approved the 23-capability inventory. No capability
removal is authorized beyond the existing later M95 entry-point replacement;
all retained behavior needs replacement evidence before Gate 5B/M95 closure.
Inventory source paths, 23 unique entries, prerequisite retirement envelopes,
documentation-memory checks and diff checks pass. Closing review stays 0/10;
this inventory approval is not milestone acceptance or Gate 5B approval.

## M91.2 selected — shared writer and elicitor review

Move `internal/icot/artifactwriter` to `internal/artifactwriter` and the coupled
elicitor/review implementation to `internal/elicitor`, preserving package names,
APIs and implementation bodies. Moving the coupled elicitor early carries its
session/discovery code into neutral ownership without copying private helpers;
M91.3 still owns removal of its transport-specific Authoring iCoT dependency.
Rebase callers and current test-command paths, fix only changed fixture-relative
paths, preserve historical harness/qualification records, and verify identical
tracked corpus/step fixture bytes. No iCoT command or user capability is removed.

## M91.2 complete — relocation equivalence

The writer and all 78 coupled elicitor files moved without copied implementation.
All 81 moved files match their `968f334` bodies after import-path and two
fixture-relative-path adjustments. All 405 tracked evaluation/step fixture files
match the pre-task digest inventory at
`/var/tmp/openudon-m91-fixtures-before.json`.

Focused writer/elicitor/step/iCoT/browser tests pass, as do transactional writer
and discovery race tests, and step optimistic-write/stale-intent/unsafe-review
race tests. `make fast` passes (full Go tests and documentation-memory checks).
Logs: `/tmp/openudon-m91-2-focused.log`,
`/tmp/openudon-m91-2-race.log`, `/tmp/openudon-m91-2-step-race.log`, and
`/tmp/openudon-m91-2-fast-fixed.log`.

The first full test detected an accidentally rewritten historical integration
selector. Its original command paths were restored; immutable v2/v3 inventory
hashes and existing report validators now pass. M91.5 owns evaluation rebasing
and any needed versioned selector; no historical report, lock, source fixture,
command capability or approval policy was changed. Final import independence,
smoke/frozen qualification and closing review remain milestone work.

## M91.3 dependency discovery — awaiting scope decision

Read-only inspection of Authoring's published HEAD
`b417eb681746476cca80bdb14cde3e3c96de980c` and OpenUdon's pinned
`2f73e3526583d303bd67a505a54035f8b1618ae0` finds the generic progressive
loop, prompt compatibility envelope and clone-based `InterviewBinding` in
`github.com/OpenUdon/authoring/icot`. Neutral interview/readiness/session/prompt
primitives exist, but no neutral public engine entry point exposes that complete
contract. `GOWORK=off go list -deps ./internal/stepauthoring` confirms the
remaining transitive `authoring/icot` dependency after M91.2.

Authoring's instructions assign generic loops, sessions and atomic bindings to
Authoring. A recommended additive prerequisite would relocate the existing
implementation into a neutral Authoring engine and keep the old `icot` API as
compatibility aliases/forwarders. Ramen still imports the old API; its source
and ledger were not changed. This is a compatibility-preserving extraction,
not the deferred Authoring iCoT/icotcli retirement.

Authoring has no pending owner for that prerequisite and is absent from this
run's execution/publication targets. Under Kinet GOAL's scope stop, M91.3 is
blocked awaiting the complete prerequisite proposal's approval and explicit
extension of the goal. No Authoring ID, planning file, code, commit or remote
was changed. M91's review count remains 0/10 and later rows were not started;
completed relocation evidence is preserved in `1a2570232cdbe8cafecda7e580f91b8a2af12746`.

## Upstream prerequisite approved

The user approved Authoring M29, extended the existing goal and authorized
Authoring publication. M91.3 remains blocked on its accepted/published exact
revision, rather than on user approval. Resume after M29 handoff and record
its source/module/qualification; retain M91.1/M91.2 and review count 0/10.

## Authoring M29 exact source reconciliation

Authoring's neutral engine passed closing review iteration 1 with no open P1/P2.
Qualified source `18056cb6b0c1007dd567a4a825a6b4311a357185` is published;
Go resolved `v0.0.0-20260930234600-18056cb6b0c1` with that exact Origin.Hash.
Producer source/qualification/review publication is independently verified at
`2a929b7686af5710db7a1600dcc1f2d27f6e388d`. Frozen qualification manifest:
`/var/tmp/authoring-m29-compat-0w5o3ztc/manifest.json`.

M91.3 now has a concrete compatible engine contract and owns exact-pin adoption
and removal of non-iCoT imports of Authoring icot. It remains pending while
M29's single execution owner completes the normal closure handoff. M91.1/M91.2
and its 0/10 review counter are preserved; no premature consumer qualification
is claimed. Ramen's old API remains supported and its source/ledger unchanged.

## M91.3 resumed after verified producer closure

Authoring M29 is complete and its closure is independently verified published
at `dc8f3d61970ae628fc0399b0ef42187aa62a3e5b`. Qualified source/module remain
`18056cb6b0c1007dd567a4a825a6b4311a357185` /
`v0.0.0-20260930234600-18056cb6b0c1`. M29 review passed 1/10; full owner,
frozen consumer, race and scorecard gates passed. M91.3 is now the sole general
in-progress row. Adopt the exact module and neutral engine aliases; preserve
consumer APIs/wire/session/fixture behavior and existing safety gates.

## M91.3 complete — published neutral engine adoption

The three internal Authoring adapters now import `authoring/engine`, preserving
all existing APIs/bodies after import/alias normalization. OpenUdon pins the
verified published module `v0.0.0-20260930234600-18056cb6b0c1`; no other module
pin changed. The module's Origin.Hash and sums were observed in
`/tmp/openudon-m91-3-authoring-module.json`.

Focused authoring/elicitor/step suites pass in both the workspace and standalone
with the actual published module (`-mod=readonly`). `make fast` passes. The
standalone step dependency closure includes engine and excludes Authoring icot
and OpenUdon internal/icot. All 405 corpus/step fixture hashes are unchanged.
Logs: `/tmp/openudon-m91-3-focused.log`, `/tmp/openudon-m91-3-standalone.log`,
`/tmp/openudon-m91-3-fast.log`, `/tmp/openudon-m91-3-deps.txt`.
M91.4/5 own remaining OpenUdon browser/evaluation import seams; closing review
stays 0/10 and no final extraction acceptance is claimed.

## M91.4 selected

Relocate browser controllers, authoring engine/application and their UI/
qualification adapters into shared packages while retaining current command
entry points, wire schemas, both capture modes and every private approval gate.
Rebase only current imports/command paths; frozen selectors/reports and legacy
session/dispatch inputs remain immutable. Extract root browser scenario/worker
helpers so non-iCoT qualification no longer imports the terminal package.

## M91.4 complete — shared browser/app implementation

Capture controllers, engine and UI/control move to `internal/browserauthor`,
`internal/authoringengine` and `internal/authoringui`. Root capture, closed worker
dispatch, attestation/staging and scenario helpers move to
`internal/browserauthoring`; retained terminal calls use thin aliases/forwarders.
No algorithm is copied. The shared staging code calls the neutral artifact
writer directly. CLI integration/route-selection tests remain in the terminal;
their bodies are preserved exactly. All 78 moved files match after package,
import, fixture-root and direct-writer adjustments. All 405 tracked corpus/step
fixtures and three embedded UI assets are unchanged.

Focused workspace/standalone tests and `make fast` pass, as do the six affected
race suites (retained terminal: 82.979s). The standalone shared scenario, worker,
engine and UI closure excludes OpenUdon `internal/icot` and Authoring `icot`.
The first fast run exposed a current release-test source path; its relocated
path is corrected and the sandbox assertion retained. Logs:
`/tmp/openudon-m91-4-shared-browser.log`,
`/tmp/openudon-m91-4-fast-fixed.log`, `/tmp/openudon-m91-4-race.log`,
`/tmp/openudon-m91-4-standalone.log`, `/tmp/openudon-m91-4-deps.txt`.

Both capture modes, TOTP, worker argv, wire/session versions and all approval
gates remain available. Frozen integration selectors/locks remain unchanged;
M91.5 owns current evaluation rebase. The affected authorized smoke and frozen
integration gate follow that rebase in M91.6. Review stays 0/10; no final
acceptance, human demonstration or runtime adoption is claimed.

## M91.5 selected

Move retained terminal/expert implementation to a neutral CLI package and keep
iCoT as a compatibility adapter during 5A. Publish `openudon authoring` expert
lint/reconcile/repair/report/variants/scorecard/replay/authoring-eval commands;
retain old report schemas and canonical command labels. Rebase current Make
evaluation and browser-system source selectors. Add a versioned integration
selector for relocated tests/dependency scans while retaining immutable v1–v4
readers, locks and named coverage. No provider run or capture removal is implied.

## M91.5 complete — expert CLI and current selectors

All 16 retained terminal/expert files move to `internal/authoringcli`, with
implementation/test bodies unchanged except the package declaration. iCoT's
public internal API is a compatibility facade. `openudon authoring` exposes
eight expert/evaluation commands and rejects UI/control/worker interaction;
existing subcommand flags, report versions and canonical legacy command labels
are preserved. Current Make evaluation targets use this new entry.

Integration evaluation v5 relocates selectors while retaining all 19 gates and
every named-test inventory. v1–v4 readers/locks remain supported and immutable;
v4 selector SHA-256 is
`737ddc4b2616d7eee34713be10d4e1781108b5e80c7caabdf8916bf9a7fdc539`.
v5 dependency scans also reject both iCoT packages. Native browser runner
selectors follow the shared UI/controller paths, preserving stage identities,
input inventory, wires and sandboxing. No source/binary lock was advanced.

Workspace/standalone CLI, browser-system, integration-validator and evaluation
tests pass; `make fast` passes. The replacement CLI actually ran the full
provider-free scorecard: 103/103 passed, zero false passes or diagnostic gaps;
its report verifies. Variants validation and all eight provider-family coverage
checks pass. The standalone OpenUdon command/step/scenario closure excludes
both iCoT packages. All 405 fixture bytes are unchanged. Logs:
`/tmp/openudon-m91-5-cli.log`, `/tmp/openudon-m91-5-selectors.log`,
`/tmp/openudon-m91-5-fast.log`, `/tmp/openudon-m91-5-scorecard.log`,
`/tmp/openudon-m91-5-standalone.log`, `/tmp/openudon-m91-5-deps.txt`.
Final smoke/frozen qualification and review/publication remain M91.6; no
provider/model, real account, desktop operation or capability removal occurred.

## M91.6 selected

Qualify clean source `2a75879c0f0a38c810877f602e90cbb9724609b8` with exact
existing compatibility/build-input sources, provider-free/offline checks and
fresh sandboxed loopback evidence. Verify availability before running; preserve
immutable locked inputs and distinguish development smoke from native
qualification. No substituted moving tips, disabled sandbox or desktop/live
operation is permitted. Persist the closing-review counter before review,
then publish/reconcile/retire only after all required gates pass.

## M91.6 qualification progress and display prerequisite

Frozen source is `2a75879c0f0a38c810877f602e90cbb9724609b8` in
`/var/tmp/openudon-m91-qualified-1y6zysvk`, with exact existing source locks and
separately copied read-only, lock-matched JavaScript modules. Frozen `make check`
and owner `go vet` pass. Sandbox-enabled Chromium readiness passes with the
locked 151.0.7922.34 runtime. A fresh `make smoke` UI stage passes (67,676ms),
with no reuse and no runtime-qualification claim. Preparatory failures (empty
ignored test directory, historical development build-input selection and
JavaScript dependency directory layout) are preserved and corrected only in
disposable inputs. Frozen compatibility locks were not modified.

Fresh current native qualification passed its UI stage, then stopped at
registration with `worker_failed`. The existing Browsertools registration and
authenticated captures intentionally launch headed Chromium; this host has
installed Xvfb but no running X display. The private diagnostic and failed
report remain in `evidence/native2-current-loopback.json*`; no pass is inferred.
The launch reference's desktop authority is explicitly restricted to M93.0 and
later consumers, so an earlier temporary M91 test display requires a narrow
authority extension. The user has been asked to approve installed Xvfb for
disposable loopback qualification with TCP disabled, temporary authentication
and automatic teardown, plus recording that exception in the coordinator
launcher. No desktop process was started, no system package installed, and
M93.0/human-visible checkpoints were not consumed.

The provider-free frozen integration matrix can proceed independently. M91.6
remains selected for those checks; full native qualification, persisted closing
review (still 0/10), publication and closure remain incomplete.

## M91.6 current integration selector correction

The frozen integration matrix observed 15 passing gates, three unrequested
optional browser checks and one failed required marker. The v4-derived
handoff selector still named the absent
`TestBrowserV10ConfigPreservesAuthenticationWithoutRegistrationAuthority`;
P08's existing replacement is
`TestBrowserModernConfigPreservesAuthenticationWithoutRegistrationAuthority`,
which actually checks authentication and registration refusal for both v10 and
v11. v5 now selects/requires that real test. Gate/named-test counts and semantic
coverage are preserved; immutable historical v1–v4 markers remain unchanged.
A regression check rejects the stale marker and requires the actual current
proof. The first frozen report remains failed evidence at source `2a75879c0f0a38c810877f602e90cbb9724609b8`;
a new clean source/qualification is required before acceptance.

The selector correction passed focused integration/handoff suites, owner
`make check` and `go vet`. A source checkpoint is needed for clean frozen
qualification; its commit records this verified correction and incomplete
M91.6 evidence, not milestone acceptance. Pending display authority remains
separate from source verification and publication.

## M91.6 independent checks finished — awaiting display authority

Corrected clean candidate `3fd40d3f874bdcf668a018550112a02cd0d02409` passed
the fresh frozen v5 integration matrix and independent report verification:
16 passing required gates, zero failures and three unrequested optional browser
checks. Evidence includes 33 authoring, 27 package, eight handoff and other
named producer/consumer tests, 157 Browserdriver tests and actual engine/UI
dependency scans. Historical v4 selector hash is still unchanged. Report:
`/var/tmp/openudon-m91-qualified-1y6zysvk/evidence/integration2-v5.json`;
invocation/log: `integration2-invocation.json` / `integration2.log`.
`partial-qualification.json` binds the candidate and retained evidence digests
and explicitly records incomplete native qualification. Earlier failed reports
and original-source development smoke remain distinct evidence.

M91.6 is now blocked only on the requested temporary Xvfb authority extension.
There is no running X server/display on the verified named host. Existing
M93.0 desktop authority is not reused early without approval. Once approved,
record the narrow launcher exception, resume this same row and perform fresh
three-pass current native qualification with TCP-disabled authenticated Xvfb
and automatic teardown; then start review at its retained 0/10 counter.
No M91 publication/acceptance/retirement or downstream implementation has
started; M93.0 and the visible human checkpoints remain pending. One execution
owner retains the run, with no general row in progress and no second audit run.

## Temporary M91 display approved — 2026-10-01

The user explicitly approved already installed Xvfb on
`vps-f7dfc687.vps.ovh.us` for M91 disposable loopback qualification, with TCP
disabled, temporary X authentication and automatic teardown. The coordinator
launcher records that exact exception separately from M93.0's later desktop
setup and human-visible checkpoints. M91.6 resumes as the single general row
in progress; review remains 0/10. No package installation or live-target
authority is added. Fresh current native qualification uses verified candidate
`3fd40d3f874bdcf668a018550112a02cd0d02409` and exact frozen dependencies.
