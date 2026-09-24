# Milestone

## UWS 1.11 real-browser qualification — M86

[M86](status-M86.md) follows completed M85 and qualifies the published stack with sandboxed real browsers. It adds a separate current scenario lock and v2 reports while preserving historical scenario verification, runs the existing 23 loopback and eight journey cases plus three Browser 1.8/1.9/v10 journeys, closes all three integration browser opt-ins, and promotes the current local suites into Make and hosted release checks. Acceptance requires clean committed source identities, actual browser execution, independently verified value-free digests, zero skipped/failed required cases, full local checks, and bounded review. Public canaries and real accounts remain outside M86.

## Current-stack browser integration qualification — M85

[M85](status-M85.md) qualifies the published UWS 1.11, Browsertools,
Browserdriver and Udon revisions through OpenUdon's full provider-free
`browser-integration-check` matrix. A new report contract selects a current
compatibility lock and requires named 1.11/Browser 1.8/1.9/v10 evidence while
the previous v1 report and scenario locks remain verifiable. M85 requires a
clean committed OpenUdon checkout, exact published siblings, a passing written
report and digest verification. Installed-browser and headed-authentication
opt-ins, historical scenario qualification, live targets and runtime adoption
are separate.
Qualification passes at clean OpenUdon `c60a2ccc2ee84553db56f7fee5be4c7cfea737d1`:
16 required gates pass, no gate fails, and three explicit browser opt-ins are
skipped. The v2 report digest verifies; M85 review iteration 1 has no open
P1/P2. See [status-M85](status-M85.md) for the exact evidence and the preserved
first failed attempt.

## UWS 1.11 and Browser 1.8/1.9 adoption — M84

[M84](status-M84.md) adopts the published UWS 1.11 contract for newly
generated workflows and adds Browser 1.8/1.9 authoring, packaging and trusted
handoff after Browsertools, Browserdriver and Udon have compatible reviewed
implementations. Existing packaged documents retain their declared versions.
This work does not rewrite W13/W16 qualification history or authorize live
browser or target actions.
Implementation and bounded review iteration 1 are complete; workspace and
standalone module checks pass against published owner revisions. The historical
browser integration matrix remains locked to its earlier qualification set;
M84 uses focused synthetic handoff tests and a local Browserdriver loopback
smoke. See [status-M84](status-M84.md) for the exact revisions and checks.

## Canonical authenticated navigation — W13.1u

E20.6 implements retained structural-query navigation with existing containment and
privacy boundaries. Publication, fresh qualification, independent bindings/cleanup
and exact retained-byte adoption pass; see [status-E20](status-E20.md).
No live capture is armed; W13.1v requires current human readiness.

## W13.1q qualified and adopted cache-compatible authoring

Published Browsertools `bbcb6ae76b5a` and OpenUdon `d96862d3797f`, W8M frozen
source `7f4920c` and tested Tofu coordination `5b8e3e035ec9` pass fresh affected
smoke and full acceptance v2: four offline stages, 39 native stages across three
repetitions, and three fresh W8M journeys. The cache regression passes in every
supervised authenticated-package repetition. Canonical verification, twenty
source/twenty-four retained runtime bindings and independent unforced cleanup of
all 5,731 identities pass. Integration review 1 has no open P1/P2.

Exact retained pass-one bytes are adopted without rebuilding; disabled preflight,
independent cleanup and adoption review 1 pass. Acceptance SHA-256:
`37983c1fd3f1fe76e195aac5e34969677fe86ed0113625a5e68ff947a2972f7e`. Earlier selections, consumed attempts,
failed local checks and unrelated edits remain preserved. W13.1n's exact live
cause and authentication outcome remain unknown. W13.1r now prepares a fresh
capture and stops for human readiness. Registration remains unsupported; W14
execution readiness is not established by authoring qualification.

This documentation closeout preserves the tested Tofu coordination pin.

## W13.1q published cache-compatibility sources

Browsertools bbcb6ae76b5a is published as
`v0.0.0-20260919093955-bbcb6ae76b5a`; downloaded module bytes match the
reviewed repair. OpenUdon d96862d3797f publishes the matching dependency and
supervised authenticated-package cache regression. Remote heads are independently
verified and focused pin/compatibility checks pass. W13.1q still requires affected
smoke and fresh complete qualification before exact-byte adoption; earlier adopted
runtime and consumed attempts are preserved. W13.1r follows at fresh human input.

## W13.1p cache-compatible authenticated authoring candidate

The renewed W13 goal authorizes the supported repair, scoped publication and
qualification/adoption before a fresh human-assisted capture. W13.1o reproduces
negative Playwright response-body sizes when a post-login page reuses cached
stylesheets/images. Plain redirects, no-store and revalidation controls pass.
This local defect does not identify W13.1n's historical request or establish
backend login success. Its consumed outcome and all prior evidence stay intact.

Browsertools A14.6 disables HTTP cache through context routing before page
creation. The route resumes unmodified Chromium requests only while the guard is
active; browser-wide Fetch still owns origin/redirect/POST admission. No request
is reissued, no body is read, and invalid sizes and exact byte/request ceilings
remain fatal. Existing optional analytics denial and other violations retain
their semantics. New loopback tests cover cached resources across main/frame/
OOPIF/popup contexts, cookies, compression, budgets and clean closure.

OpenUdon E20.5 adds a shared cacheable stylesheet to its supervised authenticated
package fixture and requires two server arrivals across login, while retaining
one POST and zero denied-origin contact. The dependency pin changes only after
Browsertools publication. Wire versions and historical readers stay unchanged.

Local candidate tests/vet, 35-case browser containment, five route controls,
focused race checks and actual application/package promotion pass. The isolated
OpenUdon check's missing-sibling setup failure is preserved; corrected remaining
checks pass. Bounded implementation review 1 passes with no open P1/P2.
W13.1q owns source publication,
matching pins, affected smoke, fresh full qualification and exact-byte adoption.
The prior qualified runtime remains selected. W13.1r prepares the next fresh
capture and stops for current desktop readiness/protected credentials. Registration
remains unsupported; no real capture is armed in this implementation task.

## Current optional-script integration — W13.1m

Published Browsertools bd89bde7542b and OpenUdon c08f9522db71, with tested Tofu
coordination snapshot ed6308ae6cb3 and W8M source commit 4268047d6e16, pass fresh
affected authentication smoke, fresh W8M consumer smoke and full acceptance v2:
four native offline stages, 39 browser stages across three native repetitions,
and three fresh W8M journeys.
Canonical verification, twenty source/twenty-four retained runtime bindings and
independent unforced cleanup of 5,384 qualification identities pass. The
frozen source trees, helpers, previous selections and consumed evidence remain
unchanged. Integration review 4 closes with no open P1/P2. Acceptance SHA-256:
`e42012538f1bf510deb85fbeaa1fbe0f9248f8895f44f3f7ee57130dee3a3225`.

Exact retained pass-one bytes are adopted without rebuilding. The retained
adapter passes disabled browser-free preflight; adoption review 1 and independent
cleanup pass. This later documentation closure leaves the tested Tofu pin and
frozen inputs unchanged.

Earlier local fixture/artifact failures and the first full qualification failure
remain preserved. That run failed in consumer journey two during overlapping
browser checks; timing suggests deadline exhaustion but the exact cause remains
unproved. The passing independent smoke, fresh serial diagnostic journey and this
fresh complete qualification are separate evidence. Heavy browser gates now run
serially without relaxed deadlines/assertions or reuse of the failed aggregate.
The second serial full run fails in native mfa-security-key authoring with
browser_failure/receive/eof. Three fresh isolated repetitions and
a fresh complete loopback stage subsequently pass. Both failed full runs remain
unqualified; the underlying native failure cause remains unresolved.

W13.1n prepares one authorized <=600-second login/read-only capture with the
exact optional-denial selector and strict non-qualifying companion. Current
desktop readiness is required before launch. No real authentication/campaign
proof follows from synthetic qualification. W13 stays open and W14 still needs
two real execution receipts; registration remains unsupported.

Earlier evidence follows.

## W13.1l optional-script denial policy

Current direction: [evolution v40](../evolution/result-v40.md).

The owner approves an explicit local policy that denies non-navigation GET/Script
requests to one exact HTTPS origin without terminating authenticated authoring.
Browsertools owns the immutable generic policy and Chromium request denial;
OpenUdon passes it through local application/worker flags. W8M alone selects
https://static.cloudflareinsights.com:443. No new origin is admitted. Zero/default
configuration and every nonmatching violation retain strict fatal behavior.
Request accounting, first failure, cancellation, approved POST budgets and
browser-wide redirect/child-frame interception remain enforced. Public browser
protocols, historical readers and portable profiles remain unchanged.

W13.1l/A14.5/E20.4 own implementation and focused review; scoped publication,
affected authentication smoke and fresh complete qualification precede adoption.
No new live capture has run. Registration remains unsupported. The W13-only goal
stops when current desktop readiness or protected human input is required.


Local W13.1l implementation review3 passes. Browsertools bd89bde7542b is
published and content-verified; OpenUdon c08f9522db71 selects its published module
without a development replacement. Browsertools full tests/vet and seven-case
loopback controls pass. OpenUdon owner packages pass across the initial run and
corrected processgroup run; vet and repository/doc checks pass. Actual
application-to-worker synthetic login/package review passes with zero denied
endpoint TCP arrivals. W8M exact-policy/strict-reader/replay/CLI tests pass.
Earlier failed checks and independent unforced cleanup remain preserved.
Scoped publication, one fresh affected smoke and full qualification precede
exact-byte adoption. No real login capture has run.

Earlier recorded state:

## Current W13.1f publication and qualification

The owner-authorized scoped publication and exact-byte adoption are complete.
Browsertools `7409ac25f161654090e3dbb97ea726b6ae46148a` and OpenUdon
`c63e32ea6fa973a5f28a207c805bcef3f59c9411` are published and remote-verified.
OpenUdon pins verified Browsertools module
`v0.0.0-20260918202946-7409ac25f161` without a development replacement.
W8M's matching locks retain the tested coordination snapshot `3251677bbdce`;
this later owner-record closeout does not change the frozen tested inputs.

Fresh affected authentication smoke, all four native offline stages, all
39 native browser stages across three repetitions and three W8M synthetic
journeys pass. Canonical verification, twenty source bindings, twenty-four
retained runtime bindings, unchanged frozen inputs and independent unforced
cleanup of 5,688 qualification identities pass. Integration review 2
and exact retained-adapter preflight/adoption review 1 pass. Exact pass-one bytes
are selected without rebuilding. Acceptance SHA-256:
`b1b5fd45909bd1e1a9081b190ba55adb61cba75c878afdb5733e1b084fce9ae6`.

The initial intermittent detached-descendant test failure remains preserved;
three focused repetitions, the affected package, and qualification's fresh full
unit/lifecycle race stages pass. Its initial cause remains unresolved. No cleanup
check is waived. Historical readers, Udon's build-input closure, previous runtime
selections, consumed captures and inherited Tofu E13/A27 edits remain preserved.
W13.1g still requires fresh exact capture authority and desktop readiness;
no live operation, registration or wider origin permission follows from this row.

Earlier diagnostic-candidate state:

## Current W13.1e rejection-diagnostic candidate

The earlier diagnostic/redirect closure passed W13.1c full qualification and
W13.1d exact-byte adoption. Its subsequent live capture failed/consumed with
policy/origin_escape, before credentials or login approval. The adopted runtime
and historical evidence remain preserved.

The owner now requests more precise diagnostics. A source-only v2 candidate
adds closed rejection boundary, resource type and origin relationship, preserving
first cause, public protocols, strict historical v1 readers and all origin/POST
bounds. Local candidate preparation/testing is authorized; no source publication,
runtime adoption or live retry is selected. W13.1e owns local verification;
W13.1f will own separately authorized publication and fresh full qualification.

Final local checks pass: Browsertools full tests/vet; OpenUdon full tests/vet,
final affected packages and repository/boundary/doc checks; closed metadata and
privacy controls; and a fresh authenticated-package smoke on `sources-3`.
All 102 final-smoke identities are independently absent without force. The
review corrected two CDP resource expectations and preserved bounded errors
through known-popup/frame identity checks. Earlier failed checks are retained.
Final bounded review 3 passes with no open P1/P2 in this local scope; the
candidate remains unpublished and unadopted. W8M essential checks also pass.
OpenUdon testing uses an explicit local replacement in a private copy; its owner
worktree keeps the old published Browsertools pin until scoped publication.
The public protocol and existing private-diagnostic direction are unchanged;
evolution was inspected and no new direction version is needed.

The owner's subsequent `git commit with splits` instruction authorizes local
commits for the reviewed work. The openudon source is committed at
`7283e5289cebc9239065f5a143a3d74b60ef48e2`. Publication, dependency-pin reconciliation,
fresh full qualification and runtime adoption remain pending. No push or live
operation is part of this commit pass.

Earlier source-publication state:

## Current authorized publication state

The owner authorizes the exact W13.1c source-publication exception. Browsertools
`3c3978f8a49f8e5ce06559a5ee38de2f4e247328` and OpenUdon
`df9299c13c08acead05393f4fc9b3794a1d06bbd` are published and remote-verified.
OpenUdon pins Browsertools `v0.0.0-20260918142842-3c3978f8a49f` without a
local replacement. Its full unit suite, vet and repository/boundary/doc checks
pass after correcting an additional current-version assertion. The first failed
pin check remains preserved. Fresh clean-source scenario validation and complete
qualification are next; the adopted runtime and consumed live attempts stay
unchanged. No runtime adoption or new target contact is authorized.

Earlier preparation state:

## E20 — Private asynchronous capture diagnostics

E20 coordinates the W13.1c diagnostic and redirect candidate. The browser-wide Chromium Fetch guard passes all 19 containment controls, the existing login-redirect fixture, native owner tests/vet and a fresh authentication smoke. Full qualification failed the clean-sibling prerequisite after seven native stages; cleanup of all 1,119 identities passes. Integration acceptance awaits authorized source commits/publication and matching pins and a fresh full run. The adopted runtime and consumed live evidence remain unchanged.

W8M W13.1c is the coordinating execution owner. Acceptance requires closed
class/privacy/malformed/terminal-path tests, one affected local browser stage,
owner checks and bounded review. Changed source publication and full consumer
qualification are separate pending integration gates. No live target or account
authority follows. Status: [E20](status-E20.md).

Earlier recorded state before the redirect repair:

E20 implements the W13.1c bounded diagnostic candidate. OpenUdon carries bounded worker/controller details into an exclusive private capture companion after event closure. Public author-session/application protocols and historical diagnostic readers remain unchanged. Focused/owner checks and one fresh local authentication smoke pass. Integration review is held by the existing authentication redirect admission defect (A14.3). Publication, complete qualification and runtime adoption remain pending; the adopted closure is unchanged.

Earlier recorded state:

W16.4i.37 publishes the diagnostic repair and passes its three focused runs plus
the containing 23-scenario stage. Its single fresh full qualification then fails
at first-repetition registration_driver after six native passes. Independent
failure verification and unforced cleanup pass; the .28d runtime stays adopted.
The proposed next step targets the synthetic initialization fixture before a
wider run. Conditional adoption and the unconsumed .34e probe remain blocked.

Earlier recorded state:

The initialization repair is published: Browserdriver 3522821, OpenUdon
d4c0a80, scoped Tofu a9b20f0 and W8M 776c20b; qualification freezes W8M d3f7957.
W16.4i.34d's single full qualification failed/consumed at repetition three's
loopback_scenarios after 33 native passes. The synthetic mfa-sms-otp authoring
controller reports worker_protocol; its exact cause remains unresolved.
registration_driver passes all five tests in each of three fresh repetitions.
Independent failure verification rejects the aggregate and confirms unforced
cleanup: all 4,886 execution and eight verifier identities are absent.

No W8M consumer journey, adoption or live probe followed. The .28d runtime
remains adopted (driver v8, diagnostics/probe v4; registration v6); the published
v9/v5 repair is not adopted. The authorized .34e probe remains unstarted behind
passing qualification/adoption and current desktop readiness. Both failed local
smokes, consumed .31/.33 probes, earlier evidence and inherited edits remain
preserved. Neither the authoring failure nor live api_loading causation is proved.

Earlier recorded context:

W16.4i.28 completes the diagnostic repair through reviewed publication,
fresh qualification and exact retained-byte adoption. The fixture-timing
successor smoke passes all four tests; the full acceptance-v2 run passes all
39 fresh native stages and three fresh W8M journeys. Independent report,
source/runtime/dependency and unforced-teardown checks pass. W16.4i.28d selects
the retained full-run pass-one bytes. The previous .24l runtime, consumed
.25/.27 probes and failed .28b smoke remain preserved.

Current probes use driver v8, diagnostics v4 and W8M probe v4; registration
remains v6. The 120-second verification phase, response matching/expiry,
submission containment, production protection and network permissions are
unchanged. Provider causation and the first smoke's exact scheduling cause
remain unresolved. Real verification/registration acceptance is still unmet;
any live probe needs its own reviewed scope and authorization.

Earlier preparation and integration context:

W16.4i.28b1/M13.21 completes the fixture timing correction and its newly
authorized smoke. All four registration_driver tests pass in 84.542 seconds,
including the v4 success/intentional-timeout and v8 observability cases.
Independent report/source/dependency checks and clean unforced teardown pass;
all 172 execution and five verifier identities are absent. Closeout review 1
has no open P1/P2. The original .28b failure remains consumed and its exact
cause unproven. Reviewed publication and one fresh complete qualification now
proceed under existing authority; .24l remains adopted until .28d passes.

Earlier preparation and failure context:

## E19 — Closed authoring diagnostics and isolated verification

W8M W16.4i.37 owns the authorized sequence after the confirmed .36 diagnostic
loss. E19.1 retains an allowlisted worker class separately from receive/drain
failures, writes strict private scenario diagnostic v2 and reads unchanged v1
evidence. Preserve public success reports, author-session v2 wire, UI privacy,
output drain, child exit and joined teardown. Browser-free controls and bounded
review precede isolated mfa-sms-otp validation and broader qualification.
W8M owns fresh launch claims, separate virtual-display preparation, publication,
conditional adoption and the already authorized human-ready live probe. Inherited
E13/A27 changes remain outside this scope. See [status-E19.md](status-E19.md).

Disposition: source implementation, focused/containing-stage verification and
publication pass. The one fresh W16.4i.37e full attempt failed/consumed at the
synthetic initialization fixture in registration_driver. Canonical failure
verification and closeout review pass; E19.2 is blocked on a focused fixture
follow-up and a passing new qualification. No adoption or live probe occurred.

## E18 — Initialization diagnostics integration

Track the approved W8M .34a-e sequence and Browserdriver M13.22 in
[status-E18.md](status-E18.md). Exact source/pins are published. The one fresh
qualification failed at repeat-three mfa-sms-otp authoring (controller/worker_protocol);
all three initialization driver stages pass. Failure evidence and cleanup verify.
Adoption and the dependent live probe remain unstarted. A focused authoring
follow-up and new reviewed qualification scope precede any further full run.

## E17 — Complete: verification observability integration

Coordinate Browserdriver M13.20 and W8M W16.4i.28. E17.1 adds the maintained
synthetic observability cases to registration_driver, then reconciles exact
published compatibility pins. One affected fresh smoke and one fresh complete
acceptance-v2 qualification are authorized by the current user goal. The W8M
ledger owns invocation claims, review, stop-on-failure and conditional adoption.
No provider service contact or live registration is authorized. E13 inherited
work remains separate. See [status-E17.md](status-E17.md).


## E16 — Complete: native input identity and repaired integration

E16.1 provides the closed native input identity; E16.3's published OpenUdon
2f95a06 repair joins authoring lifecycle completion and retains closed
failure diagnostics. E16.2 passes full downstream integration under W8M .24k:
39 fresh native stages plus three consumer journeys in v2, then explicit v3
reuse plus three fresh journeys with zero native reruns. Independent provenance,
source/runtime, cache and cleanup verification and bounded reviews pass.

W8M .24l adopts exact retained seed pass-one bytes and rebinds a disabled English
probe proposal with browser-free preflight. Historical .24g cause remains
unproven; all consumed claims, previous runtimes and inherited E13/A27 changes
are preserved. Live provider/W8M execution and real accounts are outside this
goal. Native qualification remains three fresh repeats; cache policy stays in
W8M. See [status-E16.md](status-E16.md) and [evolution v39](../evolution/result-v39.md).
Evolution v39 remains current because this closes its existing direction.

## Earlier qualification context

E15.8 published OpenUdon b3a40a9 with Udon 884a4ff and consumed W16.4i.18's
single qualification. Offline checks and 33 native stages pass; the repaired
Udon contract passes in all three repeats. The native component stopped during
repeat three's loopback scenarios at 3,600.504 seconds, consistent with the
frozen W8M wrapper's one-hour component deadline. No final native report or exact
command error was retained, so native failure verification is unavailable; the
incomplete aggregate is rejected. No W8M journey or adoption occurred. All 4,217
recorded identities are absent, and previous kits, dependency bytes, consumed
attempts and inherited E13/A27 edits remain preserved. Deadline/cancellation
evidence review precedes another separately authorized qualification. See
[status-E15.md](status-E15.md).

E15.8 prepares OpenUdon's exact Udon 884a4ff compatibility pin and matching
assertion after the test-readiness repair. Other component pins, fourteen
auxiliary build inputs, module sums and runtime contracts stay fixed. The
candidate is unpublished; browser-free checks and bounded preparation review
precede separate publication and one acceptance-v2 authorization under W8M
W16.4i.17. Existing runtime, consumed attempts and inherited E13/A27 edits remain
preserved. See [status-E15.md](status-E15.md).
Preparation checks and bounded review iteration 1 now pass; the concrete
publication/qualification proposal is ready for separate authorization.

W16.4i.11 ran the corrected qualification once. All native offline prerequisites
passed, including driver_unit, then acceptance-v2 stopped at the first repeat's
udon_browser_contract cancellation test after four passing native stages.
The test reported a grandchild PID still present; all 1,047 recorded process
identities are absent after clean teardown without forced cleanup. Native
failure evidence verifies; the incomplete aggregate is rejected for adoption.
No runtime was adopted and this invocation is consumed. The previous kit stays
selected. Sources, installed dependencies and prior evidence remain unchanged.
The supervisor's deferred orphan reaping is a leading hypothesis, not a proven
cause. A focused browser-free reproduction is the next engineering step;
any fresh full qualification needs new authority. See the latest owner status.

E15.6 coordinates Browserdriver M13.17/a probe-diagnostics publication with W8M W16.4i.8/9. Review the narrow compatibility pin, publish the scoped Tofu records, freeze sources and run one complete acceptance-v2 gate after separate authorization. Acceptance requires independent source/report/runtime/teardown verification and exact passing-byte adoption. Existing E13/A27 edits are outside this publication scope; prior failed attempts and the operating kit remain preserved.

A30.4 / E15.5 repair the September 14 English-authoring findings. iCoT retains
validated worker diagnostics through teardown and renders them separately from
controller failures. Browser-free tests/vet, focused race tests, actual package
promotion and failure UI checks pass; source review iteration 2 closes.
Published OpenUdon 36a5b5b89c2c and its exact repaired dependencies pass
39 fresh native stages, three W8M journeys and independent exact-byte adoption
under W16.4i.4. E15.5 integration review iteration 3 closes.
See [status-A30.md](status-A30.md) and [status-E15.md](status-E15.md).

## Approved verification successor — A30 / E15

The approved September 13 verification release covers Turnstile, reCAPTCHA v2
and hCaptcha, including invisible widgets activated by an approved Submit.
Background readiness advances to final approval; actual challenges remain human.
Keep production protection enabled. Detection proposes metadata; explicit review
grants bounded provider traffic separately from exact application destinations
and the one approved application POST. Tokens stay inside the browser.

Dependency order: UWS BRP/call 1.2 (tracked by W8M W16.4f) -> Browsertools A13
and Browserdriver M13 -> Udon M41 -> OpenUdon A30/E15 -> W8M W16.4g–k.
Preserve published 1.0/1.1 and UWS core. Add author-session/result v4, driver v6,
authoring-authority v2, operation-packet v3 and explicitly version expanded
review/transaction envelopes. Old consumers reject new versions before launch.

Provider policy binds HTTPS domain/path/method/frame rules and finite maxima:
256 requests, 32 MiB responses and 120 seconds, clamped by the operation deadline;
consumer authority may tighten them. Provider POSTs cannot consume or authorize
application submissions, popups, top-level navigation, downloads or other traffic.
Readiness is client evidence only; backend acceptance remains separate. Missing
or ambiguous widgets, expiry, unsupported integrations and early/duplicate POSTs
stop. Never replay a click, reload or automatically retry registration.

Each owner requires focused offline tests, one affected synthetic browser smoke,
privacy canaries and a bounded whole-diff review with no P1/P2 findings, maximum
ten iterations. Full acceptance v2 follows explicit dependency publication and
source freeze, once the integration candidate is ready; adopt exact tested bytes
and preserve the operating kit. Provider-network tests and real W8M authoring,
verification-only probing and registration require separate explicit authority.
The verification-only probe uses the trusted driver, no private inputs, zero
application mutations and clean teardown; it creates no registration claim.
Failure-report verification is distinct from successful-registration evidence.
reCAPTCHA v3/Enterprise assessments, solving services, fingerprint evasion and
production configuration changes are outside scope.

A30/E15 are complete: fresh acceptance v2, independent verification and exact
tested-byte adoption pass in W8M W16.4h. Provider-network and real-account gates
remain separate. Status: [A30](status-A30.md), [E15](status-E15.md).

E14 is complete: Browserdriver M12/Udon M40 foreground and deadline integration
passed owner checks, fresh smoke and W16 acceptance v2. Exact tested runtime
bytes are adopted; see [status-E14.md](status-E14.md). Live authority stays separate.

E13 is complete and its implementation and planning commits are published:
fast/smoke development checks, phase timing and safe explicit test/build reuse
passed unit and fresh/cached synthetic verification.
Development reports cannot become operational qualification; no new runtime
was adopted. See [status-E13.md](status-E13.md).

A29 is complete: generic registration 1.1 through iCoT, Browsertools v3 and
Udon's separate private input form passes the complete three-unit W15 consumer
gate, independent verification and exact tested-runtime adoption. See
[status-A29.md](status-A29.md). A28 discovery remains independent private state;
generated BRP/UWS files retain only reviewed public definitions and symbols.

## Memory Bank Index

- This file owns milestones, work sequencing, acceptance criteria, the current-state dashboard, and
  the status-file index.
- Use [product.md](product.md) for product scope and non-goals.
- Use [architecture.md](architecture.md) for system boundaries and planned structure.
- Use [tech-stack.md](tech-stack.md) for dependency and tooling defaults.
- Use per-milestone `status-<LANE><NN>.md` files for task-level status history.

## Status ID Pattern

Status files use one uppercase domain letter and a zero-padded number from
`01` through `99`.

Lane meanings:

- `M`: completed legacy history and future cross-cutting public contracts.
- `B`: bootstrap history only. The pre-numbered M0 record is explicitly mapped
  to B01 by M71 so it can use a canonical ID without colliding with M01.
- `A`: iCoT, workflow intent, authoring sessions, source selection, and
  authoring UX.
- `P`: synthesis, package/review/quality artifacts, approval, trusted-runner
  handoff, and release packaging.
- `E`: eval corpora, scorecards, provider drift, smoke matrices, and release
  evidence.

M71 explicitly normalizes legacy M1-M6 and M9 filenames to M01-M06 and M09
without changing their identity. M07 and M08 restore status ledgers for the
already-completed roadmap scopes. Never reuse an ID, reclassify completed
history for tidiness, or create aggregate `status.md`; cancelled files remain
with `[X]` rows.

Lane letters classify ownership rather than execution order. Independent A,
P, and E milestones may proceed together only when their sections name
non-overlapping files, resolved prerequisites, and downstream impacts. Shared
public contracts stay in M or explicitly name every cross-lane dependency.
Prefer one active implementation milestone per lane.

## Delivery Strategy

Keep OpenUdon as the public UWS authoring, review, package, and executor-handoff tool. Build in
verifiable slices: authoring and examples, deterministic synthesis, quality gates, eval and release
evidence, trusted handoff, readiness, and cross-repo compatibility. Push public workflow semantics
to `../uws`, API source metadata search/discovery/import/materialization/indexing to `../apitools`,
reusable execution to private executors such as `../udon`, and optional orchestration to
external services.

## Current State

A30/E15 are complete for reviewed registration verification. BRP/call 1.2,
authoring authority v2, transaction v4 and driver v6 integration passed fresh
acceptance v2, independent evidence verification and W8M exact tested-byte
adoption. Previous kits remain intact; provider integration and real W8M
acceptance remain separately gated. Evolution v38 records this boundary change.

[A28](status-A28.md) is complete: private registration discovery inventory and
owner review live in the shared application and iCoT; portable UWS recipes
retain published compatibility. UWS and OpenUdon source commits are published.
Live W8M operation remains separately blocked.

[M83](status-M83.md) is complete: additive private v3 registration attestation,
three complete consumer qualification units, independent verification and exact
runtime adoption pass. Completed M82 evidence and its private v2 meaning are preserved.

OpenUdon is the public UWS workflow authoring, review, package, and executor-handoff tool. The
implemented core includes:

- thin Go CLIs for OpenUdon, iCoT authoring, and the udon runner shim;
- `project.md` plus `workflows/intent.hcl` authoring;
- deterministic generation of workflow HCL, UWS YAML, expected plans, discovery, refinement,
  review, quality, and handoff artifacts;
- quality gates for project policy, API source availability, intent, data flow, workflow compilation,
  expected-plan matching, shared public UWS validation, review evidence, handoff policy, credentials, side
  effects, and secret scanning;
- eval reporting with reference policies, provider drift watch, release gates, and ignored run
  artifacts;
- adaptive iCoT v2 authoring with one active workflow boundary, unnumbered candidates,
  dependency-frontier rounds, bounded multi-family source discovery, v2 session/transcript/report
  wires, resumable incomplete drafts, complete proposal approval, and atomic source/artifact writes;
- transactional headless/UI engine mutations with typed failures, an
  optimistic bounded and streaming engine-owned workspace fingerprint set, fail-before-write
  plan validation, API v4 conditional snapshots with separate authoring and
  capture revisions, and a loopback-only polling accessible journey,
  acquisition, authoring, review, package, and handoff shell;
- one driver-free browser-profile transaction engine shared by API v4, the
  accessible loopback shell, and exact-authorized terminal NDJSON, with
  kind-specific BAP+BCP/BRP disclosure, separate review/prepare/promote/recover
  decisions, selected-package inspection, and no runtime route;
- one guided, revision-protected registration-authoring v2 wizard that keeps
  Chromium GET/HEAD-only, constructs the canonical credential-free BRP on the
  server, discloses retained structural queries for explicit review, waits for
  clean worker teardown, and then uses the ordinary transaction and package
  lifecycle without granting runtime authority;
- API-first browser fallback authoring from verified local or static-registry Browsertools profiles,
  including action/output mappings, lifecycle/digest review evidence, opaque runtime session posture,
  exact mutation approval, package inventory, and trusted Udon handoff without a membership service;
- non-executing Browsertools authoring handoff plans for missing UI-only
  profiles plus strict explicit-file consumption of reviewed guided-authoring
  results, without staging their evidence envelopes;
- explicit authenticated goal-directed Chromium authoring through a strict
  local Browsertools v2 protocol, human credential/MFA entry and exact MFA
  kind review, reduced-observation action/output/completion gates, private
  result reconstruction, and canonical UWS 1.7/1.8/1.9 profile staging without
  transferring a browser session;
- adoption of `github.com/OpenUdon/authoring` for the neutral interview graph, deterministic
  frontier, round engine, prompt/session defaults, structured JSON fallback, lifecycle atomic writes,
  and prompt-safe context records while keeping OpenUdon workflow graph construction and artifact
  lifecycle downstream;
- phase-2 adoption of Authoring report/readiness contracts as validation gates for agent,
  retention, and scorecard metadata while preserving `openudon.icot-*` JSON versions;
- v0.2 trusted-runner approval templates, per-input/self/package digests,
  unique-run staging, optional Ed25519 evidence signatures, and fully
  revalidated CLI/Docker executor handoff through `OPENUDON_EXECUTOR`;
- one immutable manifest-bound trusted-runner byte snapshot for digest,
  quality, handoff, and browser config derivation, with current-file drift
  rejection at staging and platform-qualified detached-descendant cleanup;
- production browser-workflow execution through that same trusted runner, with
  package-derived value-free driver protocol, symbolic credential/session
  mappings, exact approvals, isolated local/Docker environments, read-only
  Docker driver mounting, inert API-first fallbacks, and external runner
  revalidation;
- fail-closed browser authoring and lowering over nested session order,
  profile discriminators, symbolic bindings, strict review inputs, exact live
  origin/frame/MFA facts, and process-tree/executable safety;
- authentication-real browser evidence with session-gated fixtures,
  server-verified password/MFA behavior, clean pinned sibling revisions, and
  exact Playwright/Chromium compatibility;
- local readiness reporting and repository boundary checks;
- public MkDocs documentation with strict build in deploy automation.

OpenUdon M77 is complete. Its reconciled baseline, public
`openudon.browser-profile-transaction.v1` contract, schema, guide, examples,
product/tooling alignment, and three-iteration review gate are committed
locally. Browsertools M26 is complete at
`c52af3c058eb0f2cd021487260a9a692c67fdeb4`, and A08 is complete and published
at `39e32c1d6f601561cc5c13ec85201815ce85ab9b`
(`v0.0.0-20260825225202-39e32c1d6f60`) after its three-iteration bounded review
and sandboxed GET/HEAD-only loopback. OpenUdon A19 is complete at
`1fefb8e7672e542a6f6988f5e3cf6a09632c898b`, and P05 is complete at
`2121f06d6eab173012ab9d2a0f797ea45cece617`. OpenUdon A20 is complete at
`2b3687de2e9d1b38499e70137336a45d705e60a4` after all 23 authenticated
browser loopbacks, all 8 package journeys, and the complete sandbox-required
release gate passed. OpenUdon E10 completed at
`bb69c5a530eac303646e49a37455ce1bf19b3f57`; after explicit publication
authorization, the exact implementation and its publication-state
qualification follow-up are published through
`42767a160dc88ac18ac5d624ad9e17151ba50d77`. Its value-free 18-gate
cross-package qualification and bounded nine-iteration review pass. The
strict sequence is complete after private W8M W01 passed offline. Its reviewed
credential-free artifacts are now authorized for split commits and ordered
publication. Later
statuses remain pending and grant no browser access, registration runtime
support, deployment, or target approval.

The strict M78 -> A21 -> E11 general-platform sequence is complete. M78 adds
transaction v2 and the exact trusted registration handoff while preserving
transaction v1 and every BAP path; A21 adds guided registration-authoring v2;
and E11 qualifies the full authoring-to-runtime path at OpenUdon `f75d963` with
18/18 gates and 20 digest-linked artifacts. Coordinated evolution results
v16/v6/v14/v32 are written. Browsertools and Udon implementation commits are
published, while Browserdriver, OpenUdon, the result commits, and this memory
closeout remain local. At that sequence's close, private W8M W02 was next and
could not start until every
exact general and memory commit is published and the separate W8M target,
query, redirect, locator-metadata, and cleanup approvals are granted.

The independent UWS 1.9.1 content-trust sequence has completed and published
A22, P06, and E12. It consumes published UWS `9e676eaa469e` (UWS 1.9.1), published
Browsertools `75fd5c3ab81f` (M28), and, for E12 compatibility only, published
Udon `207e7f1` (M37). OpenUdon content-trust qualification closes at `cc378be`
and the clean-checkout CI hermeticity follow-up is published at `2c99fde`.
Hosted Actions run `33104475699` passes its public-module job and all six
release-build jobs. Content-trust analysis is advisory and does not change
trusted-runner authorization or ordinary execution behavior. At that
content-trust sequence's close, W8M W02 remained outside it and unstarted.

OpenUdon A23 is complete and published at
`dd7437c0149903ee7af987cd2e02380735ccbc40`. It adopts Browsertools A10
`3107470313d447e29c5ac5912c3cc9d221d46967`
(`v0.0.0-20260829181035-3107470313d4`) and updates the exact dependency
qualification lock. Complete offline consumer tests, vet, `make check`,
sibling/API boundary checks, and documentation-memory validation pass. This
pin-only adoption adds no public wire, browser authority, target origin, or
runtime behavior; W8M qualification remains a separate downstream gate.

OpenUdon A24 is complete and published at
`bf90537be965ace7885d43de53dab1f9eb2f2ab9` after downstream no-submit use
exposed a generic guided registration mismatch. The public UWS and Browsertools
contracts already allow one symbolic slot to be reused and accept an explicit
reviewed success condition; OpenUdon now stops presenting a current-page
locator as observed post-submit proof, makes slot reuse selectable, and includes
the common `contact_name` symbolic input. Independent `origin/main` resolution
returned the exact implementation. The remediation grants no target, submit,
account, runtime, release, or deployment authority.

OpenUdon A25 is complete and published at
`561fd933097abe90cdbe2b59b56e0fa33d4d41c1`. It began after the first downstream
A24 session exposed a false-positive credential-value classification for valid
namespaced symbolic registration bindings. Deep cross-package review then found
that intermediate commit
`e5b8a87d00ffd158dc7b71e7b5c264d320bf425f` still accepted letters-only opaque
multi-token values. The published fix uses a closed purpose-word vocabulary plus
at most one short digit-bearing namespace while preserving the reviewed
W8M-style and natural non-slot names. Downstream pin adoption and any new target
session remain separate decisions.

OpenUdon A26 is complete and published at
`e40f56fc71b9e336f270118dde9cc1830326b489`. It preserves a closed terminal
registration failure code after worker teardown and enforces exactly one
registration-authoring Launch per iCoT process independently of client activity
state. Focused/full offline gates, bounded review, and independent remote-tip
resolution pass. Downstream pin adoption, fresh preflight, and any new target
session remain separate decisions.

OpenUdon A27 is complete and published at
`552ee3a88a595cb25af1d089a0d3beeebba2a24f`; see [status-A27.md](status-A27.md).
A process-tree termination timeout outranks an ordinary worker exit, the
terminal value-free outcome survives bounded event-stream saturation, and
registration containment failure closes every later browser, package, and
authoring/transaction mutation gate. Linux process identities remain immutable
and teardown-health failure remains observable. Implementation, bounded review
and exact Browsertools A11 adoption are complete. Subsequent W8M qualification
and runtime adoption are recorded in W8M's own ledgers.

## Active And Parked Tracks

- **Completed verification:** A30 dependency review/package binding and E15
  failure-report integration passed focused checks, fresh complete qualification
  and bounded review. UWS, Browsertools A13, Browserdriver M13 and Udon M41
  are published; W8M W16.4h adopted the exact tested runtime. Live gates remain.

- **Completed cross-package sequence:** M77 -> Browsertools M26/A08 -> A19 ->
  P05 -> A20 -> E10 -> private W8M W01. OpenUdon E10 completed at
  `bb69c5a530eac303646e49a37455ce1bf19b3f57` and is now published through the
  publication-state qualification follow-up
  `42767a160dc88ac18ac5d624ad9e17151ba50d77`; Browsertools A08 and UWS remain
  published at their exact locks. W01 passed offline without target access,
  browser launch, account action, workflow execution, or deployment. Evolution
  result v31 records the completed run and its subsequent publication.

- **Latest completed cross-package milestone:** M78 -> A21 -> E11 general
  browser-registration platform. The value-free qualification passes 18/18
  with 20 artifacts, one approved synthetic POST, and no registration session;
  all BAP, browser, UI, race, audit, secret, strict-doc, and release gates pass.
  Evolution results v16/v6/v14/v32 are recorded locally. A separately
  authorized private W8M W02.3 no-submit session subsequently occurred and
  stopped before candidate adoption; any additional session remains gated on
  fresh exact-pin preflight and separate one-session approval.

- **Completed content-trust sequence:** A22 added operator declarations and
  conditional UWS 1.9.1 synthesis, P06 added warning-only quality/review
  evidence, and E12 passed the offline matrix plus exact published Udon M37
  compatibility. The sequence and its clean-checkout CI follow-up are published
  through OpenUdon `2c99fde`, and hosted CI is green. No task granted runtime
  authority or advanced W8M.

- **Completed Browsertools A10 consumer adoption:** A23 pins the published
  nonfatal denied-subresource remediation and its exact content-trust
  qualification expectation. OpenUdon `dd7437c` is published after all
  offline consumer gates passed. No registration session or target contact
  was part of the adoption.

- **Completed authoring remediation:** A24 makes guided registration
  slot mapping and post-submit proof honest. It is OpenUdon-only: Browsertools
  continues to bind the observed submit candidate, while the operator-reviewed
  success locator is explicitly unobserved during no-submit authoring and
  deferred to runtime proof. OpenUdon commit `bf90537` is published at exact
  `origin/main`. See [status-A24.md](status-A24.md).

- **Completed authoring remediation:** A25 closes A24's symbolic-binding
  false-positive without widening credential-value acceptance. Deep review
  found and the published `561fd93` fix remediates a letters-only opaque-token
  bypass in the first local commit. Its scope is local validation, regression
  coverage, documentation, and offline checks; it grants no browser, target,
  submission, transaction, runtime, downstream pin, release, or deployment
  authority. See [status-A25.md](status-A25.md).

- **Completed authoring remediation:** A26 makes the single Launch
  boundary server-authoritative and retains a closed value-free terminal
  failure class. Its scope is additive experimental API state, UI locking,
  regression coverage, documentation, and offline checks. OpenUdon
  `e40f56fc71b9e336f270118dde9cc1830326b489` is published at exact
  `origin/main`; downstream pin adoption and any new browser or target session
  remain separate. See [status-A26.md](status-A26.md).

- **Latest completed milestone:** A01 API-first browser-profile fallback authoring. Matching API
  operations remain preferred; verified Browsertools profiles now cover UI-only gaps through
  digest-bound review packages and trusted Udon handoff. No membership service, browser session,
  credential, driver, capture, or execution behavior moved into OpenUdon.

- **Latest completed milestone:** A02 additive browser authentication authoring. OpenUdon now
  discovers local secret-free authentication profiles, authors one explicit
  sign-in/MFA flow with symbolic credentials and an execution-local named
  session, lowers the reviewed intent to UWS 1.7 supplements, and binds safe
  profile/approval evidence into package quality and trusted handoff. Udon and
  Browserdriver retain all credentials, challenges, live sessions, and
  execution behavior.

- **Latest completed milestone:** A03 Browsertools authoring handoff. iCoT emits
  an explicit, non-executing external authoring plan and can consume a reviewed
  guided-authoring result without launching a browser or absorbing private
  evidence. See [status-A03.md](status-A03.md).

- **Latest completed milestone:** A04 explicit authenticated browser
  authoring. `icot browser-author live` now coordinates Browsertools' one
  headed Chromium context across human login/MFA and post-login goal
  exploration, consumes only reduced semantics, independently validates the
  private result, and stages canonical UWS profiles after final approval. See
  [status-A04.md](status-A04.md).

- **Latest completed milestone:** A05 live browser observation hardening.
  OpenUdon now checks every incoming label with Browsertools' canonical E06
  reducer, emits only safe closed rejection diagnostics, enforces the requested
  128-candidate/match bound before disclosure, and keeps typed action validation
  as the primary planner boundary. See [status-A05.md](status-A05.md).

- **Latest completed milestone:** A06 typed MFA and dashboard output
  authoring. OpenUdon now consumes only Browsertools author-session/result v2,
  keeps compatible MFA and bounded output selection human-only, reconstructs
  returned profiles before atomic staging, and selects UWS 1.9 for browser 1.7
  scalar accessibility outputs. See [status-A06.md](status-A06.md).

- **Latest completed milestone:** A07 headless iCoT authoring engine. An
  internal driver-agnostic engine now exposes JSON-marshalable snapshots,
  complete frontier-round application, preview, resumable autosave, and an
  explicit human-approval write boundary while sharing the terminal path's
  source revalidation and atomic artifact transaction. It adds no UI, server,
  or public schema. See [status-A07.md](status-A07.md).

- **Latest completed milestone:** A08 local iCoT UI server. `icot ui` adds a
  single-workspace, loopback-only experimental JSON transport over A07 plus an
  embedded read-only status shell, with per-process token authentication,
  exact revision checks, and frozen post-write inspection. See
  [status-A08.md](status-A08.md).

- **Latest completed milestone:** A09 enhanced Phase B reliability and status
  UX. Engine mutations now have prospective transactional snapshots, typed
  failure classes, revision-bound workspace fingerprints, and hardened
  rollback/symlink handling. The loopback transport is API v2 only, with
  conditional snapshots, strict UTF-8/duplicate-safe JSON, request resource
  limits, safe error IDs/logging, and a polling read-only shell. See
  [status-A09.md](status-A09.md).

- **Latest completed milestone:** A10 interactive Phase C authoring and review
  shell. The current frontier is now an accessible complete-round form with
  revision protection, preview/action/readiness/conflict review, separate final
  and incomplete approval controls, and explicit stale/drift/retry/freeze
  reconciliation without new production frontend dependencies. Phase C review
  closure also rejects ambiguous/reserved write plans before mutation, bounds
  candidate observation and hashing, and restores successful-mutation focus
  plus polite status announcements. See
  [status-A10.md](status-A10.md).

- **Latest completed milestone:** A11 Phase C real-browser qualification. A
  build-tagged Playwright-Go/Chromium suite exercises the actual loopback
  listener, accessible keyboard journeys, both approvals, conflict and failure
  states, polling/visibility/304 behavior, and narrow/zoom layout. All 13
  journeys pass with the release target requiring and logging Chromium sandbox
  use; the disable override is rejected. See [status-A11.md](status-A11.md).

- **Latest completed milestone:** P02 trusted execution and package integrity.
  Executable v2 handoffs bind exact config, approval, handoff, package, and
  executor-report bytes; typed invocations isolate environments; v1 execution
  is rejected; and optional signatures support trusted operator keys. See
  [status-P02.md](status-P02.md).

- **Latest completed milestone:** A12 authoring, provider, and UI safety.
  Credential detection is shared, Gemini and remote-source transports are
  bounded, the browser opens without a secret, source roots are canonical,
  draft state clones deeply, security choices survive reordering, and HCL
  diagnostics retain original lines. See [status-A12.md](status-A12.md).

- **Latest completed milestone:** A13 iCoT UI review remediation. Accepted
  answers are effective and field-addressable; engine-owned controls expose
  closed choices and deferral; eligible settled decisions support bounded
  exact-revision correction; browser access is recoverable; and approval shows
  safe evidence without widening the CLI-first, single-operator, non-executing boundary. See
  [status-A13.md](status-A13.md).

- **Latest completed milestone:** E06 honest and reproducible evidence.
  All-skipped browser runs report `not_run`, subprocess trees have fixed
  deadlines, persisted paths are portable, eval workspaces self-clean, map
  output is deterministic, and Slack fixtures are minimal. See
  [status-E06.md](status-E06.md).

- **Latest completed milestone:** M74 v0.2 consolidation and closure. Shared
  atomic/evidence helpers, extracted CLI policy, zero pinned dead code, deleted
  schema snapshots, responsibility-split source files, and documentation leave
  the tree ready for release evidence without creating a tag. Its formerly
  pending A11.5 sandbox proof is now closed by E09. See
  [status-M74.md](status-M74.md).

- **Latest completed milestone:** P03 trusted browser execution. Normal
  `openudon run` now derives and binds the complete value-free Udon/
  Browserdriver invocation contract from the reviewed package, admits only
  declared browser credentials and launcher state, and makes the external
  runner prove the same contract before execution. See
  [status-P03.md](status-P03.md).

- **Latest completed milestone:** A14 browser authoring and synthesis review
  closure. Live observation, approvals, origin inventory, MFA families,
  process ownership, profile/version lowering, symbolic bindings, nested
  session ordering, registry text, and atomic staging now fail closed under
  shared validators. See [status-A14.md](status-A14.md).

- **Implementation complete:** A15 engine acquisition lifecycle. Journey
  starters, a bounded private API-source inbox, digest-owned staging/removal,
  append-only v3 browser-review collections, deterministic v2 migration, and
  create-only multi-profile capture staging refresh discovery transactionally.
  Its hardening follow-up unifies discovery/registry/verification refresh for
  every authoring surface and scopes session/approval decisions to every
  browser step without weakening exact-byte workspace ownership. See
  [status-A15.md](status-A15.md).

- **Latest completed milestone:** A16
  unified iCoT UI and package handoff. Experimental API v3 and the accessible
  shell cover acquisition, isolated bundled browser capture, independently
  consumed revisions, authoring approval, deterministic package build/current-
  byte assessment, failure resume/reapproval, drift-invalidated frozen
  handoff, and closed-allowlist inspection. The hardening follow-up adds
  process-private parent attestation, bounded disclosure-path validation,
  atomic credential answers, and content-addressed worker caching. Its hardened
  Browsertools pin is published and passes fresh proxy-only resolution plus
  standalone build, test, and vet. See
  [status-A16.md](status-A16.md).

- **Latest completed milestone:** E07 authenticated browser evidence. The
  deterministic loopback proves server-observed password and every MFA replay;
  scenario and integration gates reject dirty or mismatched pinned siblings,
  while scenario preparation enforces actual Playwright/Chromium versions. See
  [status-E07.md](status-E07.md).

- **Latest completed milestone:** E08 unified UI acquisition-to-handoff
  qualification. Provider-free API, protocol, lifecycle, acquisition,
  multi-profile, process, terminal-compatibility, package-recovery, drift, and
  route-closure tests pass. The compositional leak proof combines those exact
  API/engine/process surfaces with 13 real sandbox-required Chromium UI
  journeys, and the published Browsertools A06 baseline passes cold-cache
  standalone verification. New adversarial trace/path/origin/limit coverage is
  green in the coordinated workspace; fresh proxy-only resolution, standalone
  build/test/vet, full tests/vet, iCoT race, repository checks, browser
  integration, and the headless journey matrix pass. Real-browser reruns
  require a host with sandbox-compatible user namespaces. See
  [status-E08.md](status-E08.md).

- **Latest completed milestone:** P04 immutable trusted execution and containment
  closure. One byte snapshot owns all post-validation derivation, staging
  rejects later drift, Docker/outer-runner environment authority is narrowed,
  and Linux detached descendants are identity-tracked and swept after normal
  exit or cancellation. Focused/full/race, release, standalone, cross-build,
  and browser evidence gates pass. See [status-P04.md](status-P04.md).

- **Latest completed milestone:** A17 acquisition, synthesis, and UI remediation
  closure. Observation-before-read acquisition, strict legacy migration,
  stable executable/profile bytes, shared nested effective-source traversal,
  path-free doctor state, preflight rollback, and JSON depth limits now fail
  closed. See [status-A17.md](status-A17.md).

- **Latest completed milestone:** A18 secret-safe browser registration
  packaging. OpenUdon consumes the published UWS and Browsertools registration
  contracts, validates and packages exact reviewed sources, lowers explicit
  registration intent, and qualifies approval plus dry-run handoff while
  failing closed before unsupported runtime execution. See
  [status-A18.md](status-A18.md).

- **Latest completed milestone:** E09 typed and clean-root evidence closure.
  Strict Udon report v2 classification, server-observable approval MFA,
  synchronized loopback state, mandatory sandbox assertions, early standalone
  CI builds, and the published compatibility set are complete. Aggregate
  release, cold-cache standalone, clean-root browser matrices, real sandboxed
  Chromium, strict docs, race/deadcode, and cross-build gates pass. See
  [status-E09.md](status-E09.md).

- **Latest completed milestone:** M75 browser review remediation closure. The
  Browsertools M25 producer contract is pinned, shared atomic and repository
  policies are consolidated, and operator/release documentation describes the
  executable browser handoff without tagging v0.2.0. Its formerly pending
  A11.5 sandbox proof is now closed by E09. See
  [status-M75.md](status-M75.md).

- **Latest completed milestone:** M76 browser execution regression closure.
  Live interruption and failed protocol sessions terminate Browsertools'
  complete process group, interactive stdout drains before reaping, Docker
  receives the configured driver through a read-only translated mount,
  API-first fallbacks remain inert, and collision-renamed registry documents
  are rebuilt from their final plans. See [status-M76.md](status-M76.md).

- **Latest completed milestone:** M77 browser-authoring transaction contract.
  OpenUdon now publishes a value-free transaction schema and guide over the
  existing BAP/BCP/BRP families, with strict canonical encoding, immutable
  review provenance, prepare/promotion boundaries, typed recovery, and no new
  UWS or runtime authority. See [status-M77.md](status-M77.md).

- **Latest completed milestone:** P01 value-free browser verification
  packaging. Optional live/portability facts are now independently validated
  and digest-bound through package review without admitting private evidence or
  turning portability into a runtime requirement. See
  [status-P01.md](status-P01.md).

- **Latest completed milestone:** E01 browser authoring-to-handoff integration
  evaluation. Its strict, digest-bound matrix established the A03/P01
  pre-live baseline across all five participating repositories; E02 now extends
  that contract with explicit authenticated authoring and context replay. See
  [status-E01.md](status-E01.md).

- **Latest completed milestone:** E02 authenticated authoring and trusted
  replay qualification. The provider-free matrix now binds the clean A04,
  Browsertools author-session, UWS 1.8, Udon v3, and Browserdriver v3 revisions,
  while preserving browser-free defaults and explicit headed opt-ins. See
  [status-E02.md](status-E02.md).

- **Latest completed milestone:** E03 exact authenticated-authoring seam qualification.
  It closes the review findings in the live protocol, pins the coordinated
  feature commits, and makes real producer/consumer/replay tests mandatory.

- **Latest completed milestone:** E04 complementary real-browser scenario
  evaluation. Its original 21-case deterministic loopback exercises the
  production author-session v2 through Udon/Browserdriver v3; E08 hardening
  adds path-injection and fabricated-trace rejection for a current 23-case
  suite. Four fixed anonymous public canaries remain an explicit-network
  informational suite. Post-publication review closed hosted-runner sandbox
  provisioning, recursive duplicate-key decoding, and false-presence runtime
  alignment.
  See [status-E04.md](status-E04.md).

- **Latest completed milestone:** E05 realistic Playwright-browser journey
  qualification. Eight required headless local cases now carry reviewed
  Browsertools guided bundles through strict OpenUdon import, ordered UWS 1.8
  synthesis, Udon/Browserdriver v3 replay, exact outputs, approval failures,
  and server-state postconditions. See [status-E05.md](status-E05.md).

- **Completed milestone:** M71 parallel-lane harness migration. All
  historical task ledgers now use canonical IDs and runner-compatible state
  tables; future authoring, package, and eval work has explicit ownership.

- **Completed milestone:** M72 structured API security alternatives. OpenUdon
  preserves OR-of-AND authentication sets, requires one resume-safe selection
  before request mappings, and keeps prompt-budget loss as a visible technical
  deferral rather than approving a partial source interpretation.

- **Completed milestone:** M73 shared interview settlement and
  lifecycle-ranking boundary. OpenUdon now delegates complete-frontier
  projection and settlement to Authoring's clone-based binding and consumes
  lifecycle-role ranking from Apitools operation summaries. Workspace and
  published-pin standalone tests/vet pass without local replacements.
  See [status-M73.md](status-M73.md).

- **Latest completed milestone:** M53 shared Authoring iCoT consumption. OpenUdon now delegates
  generic progressive-loop plumbing to `github.com/OpenUdon/authoring/icot`
  while keeping prompts, intent schema, reports, artifacts, and provider
  clients downstream. See [status-M53.md](status-M53.md).
- **Latest completed milestone:** M54 async evidence sidecar forwarding. `openudon run` now writes
  one neutral `openudon.async-evidence-bundle.v1` sidecar per run, references it from
  `openudon.run-evidence.v1` by digest, and uses `github.com/OpenUdon/evidence/async` request and
  response records for package/run handoff audit only. OpenUdon does not interpret Ramen
  convergence, desired hashes, resource state, or provider-specific async polling. See
  [status-M54.md](status-M54.md).
- **Latest completed milestone:** M55 run evidence usability hardening. `openudon run` now prints
  the async sidecar path, run-evidence sidecar refs are workdir-relative for archive use, external
  runner failure has direct async evidence coverage, and review-handoff docs show compact sidecar
  examples. See [status-M55.md](status-M55.md).
- **Latest completed milestone:** M56 run evidence verification and release readiness. OpenUdon now
  has `openudon run-evidence verify --file run-evidence.json`, strict async sidecar validation,
  a machine-readable sidecar schema, and a passing M54/M55 release-readiness baseline. Executor
  status/read ingestion remains deferred until a real trusted-executor output contract exists. See
  [status-M56.md](status-M56.md).
- **Latest completed milestone:** M57 trusted executor output contract, M58 run evidence verifier
  hardening, and M59 async evidence release-candidate pass. OpenUdon now forwards real
  udon-owned execution reports into async status/read observations, verifies expanded sidecars, and
  documents release evidence for archives. See [status-M57.md](status-M57.md),
  [status-M58.md](status-M58.md), and [status-M59.md](status-M59.md).
- **Latest completed milestone:** M60 executor-report archive command, M61 release-note generation,
  M62 local udon executor smoke, and M63 verifier schema polish. OpenUdon can archive run evidence
  bundles, draft local release evidence notes, run a provider-free non-dry-run sibling udon proof,
  and schema-check emitted async sidecars. See [status-M60.md](status-M60.md),
  [status-M61.md](status-M61.md), [status-M62.md](status-M62.md), and
  [status-M63.md](status-M63.md).
- **Latest completed milestone:** M64 release evidence workflow consolidation.
  `openudon release-evidence` and `make release-evidence` now turn the M60-M63
  archive, verifier, release-note, and local-udon-smoke helpers into one
  repeatable local operator flow with compact JSON/Markdown summaries. See
  [status-M64.md](status-M64.md).
- **Latest completed milestone:** M65 iCoT M29 follow-up closure. Review repair
  now handles nested/ref/array response-field metadata, can insert a narrowly
  proven read-only API prework step, and has an explicit focused replay repair
  gate.
  See [status-M65.md](status-M65.md).
- **Latest completed milestone:** M66 provider expansion review. Broad
  provider-adapter expansion remains parked in OpenUdon because desired-state
  conversion and provider/resource operation mapping now belong in Ramen.
  OpenUdon may review/package UWS-facing artifacts generated elsewhere.
  See [status-M66.md](status-M66.md).
- **Latest completed milestone:** M67 desired-state conversion removal.
  OpenUdon removed old conversion docs, conversion status files, conversion
  navigation, and the retired conversion evolution record. OpenUdon keeps only
  boundary guards that prevent parser/conversion code from returning. See
  [status-M67.md](status-M67.md).
- **Latest completed milestone:** M68 iCoT lifecycle planning alignment.
  Draft rounds now derive lifecycle hints and source-correct sibling detail
  refs from one expansion plan, fall back to ranked seeds for first drafts,
  preserve bounded detailed prompt context, and keep requested detail refs out
  of lifecycle seed selection. The cross-repo review also restored M34's
  policy-backed hermetic eval discovery gate. See
  [status-M68.md](status-M68.md).
- **Latest completed milestone:** M69 CLI-first public beta. OpenUdon now has a defined
  v0.1 CLI/artifact compatibility boundary, standalone public dependencies,
  three-command release archives, local build metadata, and tag-driven release
  automation. Clean-archive and local gates passed, public CI is green, and
  v0.1.0 is published with six verified platform archives plus checksums. See
  [status-M69.md](status-M69.md).
- **Latest completed milestone:** M70 adaptive evidence-grounded iCoT v2. OpenUdon now selects one
  active workflow from broad requests, asks dependency-ready frontiers without a fixed breadth
  ceiling, consumes apitools' bounded multi-family local discovery, gates bounded remote hints,
  stages sources only after full proposal approval, supports incomplete draft promotion, emits v2
  sessions/transcripts/reports/replay/variants, and keeps agent mode read-only. See
  [status-M70.md](status-M70.md).
- **UWS 1.4 source-family coverage:** GraphQL, OpenRPC, gRPC/protobuf, and OData are now supported
  for reviewed local source artifacts, package/quality/handoff evidence, and iCoT local discovery.
  Protocol execution remains trusted-executor-owned.
- **Completed iCoT natural-language authoring reliability expansion:** M40 expanded provider-free
  iCoT reliability evidence. See [status-M40.md](status-M40.md).
- **Completed iCoT request mapping repair:** M39 iCoT request mapping repair and decision evidence. See
  [status-M39.md](status-M39.md).
- **Completed iCoT scorecard and agent mode:** M38 added noninteractive agent authoring, structured
  lint reports, and provider-free scorecard evidence. See [status-M38.md](status-M38.md).
- **Completed product smoke release readiness:** M37 added the product smoke matrix, full live Slack
  and weather smoke evidence, and `v0.1.2-a.1` tag readiness. See [status-M37.md](status-M37.md).
- **Completed Slack smoke and v0.1.2 tag gate:** M36 proved the manual Slack sandbox handoff before
  the v0.1.2 tag. See [status-M36.md](status-M36.md).
- **Completed v0.1.2 provider-free release readiness:** M35 made the release command set,
  diagnostics, docs, dry-run handoff lane, package hygiene, and compatibility evidence ready before
  the manual Slack smoke/tag gate. See [status-M35.md](status-M35.md).
- **Completed eval seed/build matrix:** M34 made the eval corpus an explicit offline
  iCoT-seed/build contract and graduated advisory n8n reducibility fixtures to build-clean
  package-local evidence while keeping them advisory. See [status-M34.md](status-M34.md).
- **Completed runtime data-file handoff:** M33 added reviewed package data-file inputs, env-marker
  placeholders, source security binding names, and trusted-runner datafile handoff. See
  [status-M33.md](status-M33.md).
- **Completed fnct helper contract authoring:** M32 added public helper selector authoring for
  `gmail.render_raw` without moving execution into OpenUdon. See [status-M32.md](status-M32.md).
- **Completed build from intent hardening:** M30 made `openudon build` reliable
  from reviewed `intent.hcl`, including native API source parity and build-only
  regression coverage. See [status-M30.md](status-M30.md).
- **Completed iCoT flow remediation and draft expansion:** M29 added gap
  taxonomy, bounded review-repair expansion, safer catalog handling, focused
  replay closure, and no-source gap-report fallback behavior. See
  [status-M29.md](status-M29.md).
- **Completed iCoT decision evidence and repair-loop evaluation:** M28 added public decision
  evidence, confidence-driven prompting, bounded review-repair metrics, and the representative
  replay corpus baseline. See [status-M28.md](status-M28.md).
- **Completed UWS 1.2 first-class API source types:** M27 added typed API source integration for
  OpenAPI, Google Discovery, and AWS Smithy JSON. See [status-M27.md](status-M27.md).
- **Completed iCoT catalog planning and operation-detail mapping:** M26 keeps the catalog-backed
  SaaS authoring path and request-mapping draft loop as the baseline for typed API source work. See
  [status-M26.md](status-M26.md).
- **Completed provider catalog CLI integration:** M25 exposes selected
  `apitools/catalog` provider inspection and direct OpenAPI import commands through OpenUdon while
  keeping catalog metadata advisory and local OpenAPI package inputs authoritative. See
  [status-M25.md](status-M25.md).
- **Completed release gate consolidation:** M24 added comprehensive provider-free
  `make release-saas-check` target while keeping `make release-check` as the fast deterministic
  pre-tag gate, and verified the target through local SaaS release evidence. See
  [status-M24.md](status-M24.md).
- **Completed SaaS operator release readiness:** M23 added the provider-free SaaS operator release
  path, selected Gmail audit receipt and order fulfillment chain demos, tightened boundary language,
  and confirmed both demos through sandbox trusted-runner dry-run smoke. See
  [status-M23.md](status-M23.md).
- **Completed n8n pattern bridge:** M22 added the `openudon.n8n-pattern-summary.v1` authoring
  evidence contract, selected three n8n-derived bridge summaries, documented unsupported n8n
  semantics, and added deterministic `openudon n8n-bridge validate` checks. See
  [status-M22.md](status-M22.md).
- **Completed strict SaaS corpus expansion:** M21 promoted trial-backed strict package coverage,
  added weather authoring metadata, normalized bearer security request placement, and confirmed
  promoted strict fixtures through sandbox dry-run gates. See [status-M21.md](status-M21.md).
- **Completed end-to-end SaaS authoring trials:** M20 selected eight realistic SaaS trials, ran
  authoring lint, package/dry-run checks where possible, documented the trial/gap matrix, and chose
  promotion candidates. See [status-M20.md](status-M20.md).
- **Completed post-M19 SaaS sequence:** M20 trials, M21 strict corpus expansion, M22 n8n pattern
  bridge, and M23 operator release readiness are complete.
- **Completed SaaS review and trusted-handoff confidence:** M19 added gated review evidence for
  side-effect risk, credential scope, approval artifacts, package digest expectations, dry-run
  validation, and trusted-runner run-config boundaries. See [status-M19.md](status-M19.md).
- **Completed multi-service SaaS workflow patterns:** M18 selected strict multi-service fixtures,
  documented pattern coverage, hardened fixture metadata, and checked selected iCoT lint gates. See
  [status-M18.md](status-M18.md).
- **Completed guided SaaS authoring UX:** M17 tightened iCoT prompts, readiness repair feedback,
  side-effect defaults, strict fixture lint checks, and public authoring docs. See
  [status-M17.md](status-M17.md).
- **Completed SaaS corpus hardening:** M16 selected the initial strict native set and kept broader
  selected-service coverage advisory or native-pattern based until graduation. See
  [status-M16.md](status-M16.md).
- **Completed SaaS sequence:** M17 guided SaaS authoring UX, M18 multi-service SaaS workflow
  patterns, and M19 SaaS review and trusted-handoff confidence.
- **Stable boundary:** OpenUdon may review UWS-facing artifacts generated
  elsewhere, but desired-state conversion and provider/resource operation
  mapping belong to Ramen.

## Status Files

| Milestone | Status File | Summary |
|---|---|---|
| B01 | [status-B01.md](status-B01.md) | Project scaffold and initial direction. |
| M01 | [status-M01.md](status-M01.md) | Post-POC baseline. |
| M02 | [status-M02.md](status-M02.md) | Eval corpus and reference discipline. |
| M03 | [status-M03.md](status-M03.md) | Structured output and provider drift. |
| M04 | [status-M04.md](status-M04.md) | Quality gate hardening. |
| M05 | [status-M05.md](status-M05.md) | Workflow artifact power. |
| M06 | [status-M06.md](status-M06.md) | iCoT authoring. |
| M07 | [status-M07.md](status-M07.md) | Safety and trusted execution. |
| M08 | [status-M08.md](status-M08.md) | Local checks and release process. |
| M09 | [status-M09.md](status-M09.md) | Converted package integration. |
| M10 | [status-M10.md](status-M10.md) | Cross-repo dependency stewardship. |
| M11 | [status-M11.md](status-M11.md) | Public OpenUdon package boundary. |
| M12 | [status-M12.md](status-M12.md) | Package artifact and local OpenAPI safety hardening. |
| M15 | [status-M15.md](status-M15.md) | Agentic UWS authoring for common SaaS workflows. |
| M16 | [status-M16.md](status-M16.md) | SaaS authoring corpus hardening. |
| M17 | [status-M17.md](status-M17.md) | Guided SaaS authoring UX. |
| M18 | [status-M18.md](status-M18.md) | Multi-service SaaS workflow patterns. |
| M19 | [status-M19.md](status-M19.md) | SaaS review and trusted-handoff confidence. |
| M20 | [status-M20.md](status-M20.md) | End-to-end SaaS authoring trials. |
| M21 | [status-M21.md](status-M21.md) | Strict SaaS corpus expansion. |
| M22 | [status-M22.md](status-M22.md) | n8n pattern bridge. |
| M23 | [status-M23.md](status-M23.md) | SaaS operator release readiness. |
| M24 | [status-M24.md](status-M24.md) | Release gate consolidation and evidence automation. |
| M25 | [status-M25.md](status-M25.md) | First-class provider catalog CLI integration. |
| M26 | [status-M26.md](status-M26.md) | iCoT catalog planning and operation-detail mapping pipeline. |
| M27 | [status-M27.md](status-M27.md) | UWS 1.2 first-class API source type integration. |
| M28 | [status-M28.md](status-M28.md) | iCoT decision evidence and repair-loop evaluation. |
| M29 | [status-M29.md](status-M29.md) | iCoT flow remediation and draft expansion. |
| M30 | [status-M30.md](status-M30.md) | Build from intent hardening. |
| M31 | [status-M31.md](status-M31.md) | Trusted execution hardening. |
| M32 | [status-M32.md](status-M32.md) | Fnct helper contract authoring. |
| M33 | [status-M33.md](status-M33.md) | Runtime data-file handoff. |
| M34 | [status-M34.md](status-M34.md) | Eval seed/build matrix. |
| M35 | [status-M35.md](status-M35.md) | v0.1.2 release readiness. |
| M36 | [status-M36.md](status-M36.md) | Slack smoke and v0.1.2 tag gate. |
| M37 | [status-M37.md](status-M37.md) | Product smoke matrix and v0.1.2-a.1 readiness. |
| M38 | [status-M38.md](status-M38.md) | iCoT scorecard and agent mode. |
| M39 | [status-M39.md](status-M39.md) | iCoT request mapping repair and decision evidence. |
| M40 | [status-M40.md](status-M40.md) | iCoT natural-language authoring reliability expansion. |
| M41 | [status-M41.md](status-M41.md) | UWS 1.3 contract baseline for AsyncAPI source descriptions. |
| M42 | [status-M42.md](status-M42.md) | AsyncAPI metadata boundary. |
| M43 | [status-M43.md](status-M43.md) | OpenUdon package and generation support for AsyncAPI. |
| M44 | [status-M44.md](status-M44.md) | iCoT, examples, and eval coverage for AsyncAPI. |
| M45 | [status-M45.md](status-M45.md) | Release readiness and documentation for UWS 1.3. |
| M46 | [status-M46.md](status-M46.md) | UWS 1.4 source-family eval coverage. |
| M52 | [status-M52.md](status-M52.md) | Ramen/OpenUdon approval and governance contract review. |
| M53 | [status-M53.md](status-M53.md) | Shared Authoring iCoT consumption. |
| M54 | [status-M54.md](status-M54.md) | Async evidence sidecar forwarding. |
| M55 | [status-M55.md](status-M55.md) | Run evidence usability hardening. |
| M56 | [status-M56.md](status-M56.md) | Run evidence verification and release readiness. |
| M57 | [status-M57.md](status-M57.md) | Trusted executor output contract. |
| M58 | [status-M58.md](status-M58.md) | Run evidence verifier hardening. |
| M59 | [status-M59.md](status-M59.md) | Async evidence release-candidate pass. |
| M60 | [status-M60.md](status-M60.md) | Executor-report archive command. |
| M61 | [status-M61.md](status-M61.md) | Release-note generation. |
| M62 | [status-M62.md](status-M62.md) | Real executor smoke with local udon. |
| M63 | [status-M63.md](status-M63.md) | Verifier schema polish. |
| M64 | [status-M64.md](status-M64.md) | Release evidence workflow consolidation. |
| M65 | [status-M65.md](status-M65.md) | iCoT M29 follow-up closure. |
| M66 | [status-M66.md](status-M66.md) | Provider expansion review. |
| M67 | [status-M67.md](status-M67.md) | Desired-state conversion removal. |
| M68 | [status-M68.md](status-M68.md) | iCoT lifecycle planning alignment. |
| M69 | [status-M69.md](status-M69.md) | CLI-first public beta and first public release. |
| M70 | [status-M70.md](status-M70.md) | Adaptive evidence-grounded iCoT v2. |
| M71 | [status-M71.md](status-M71.md) | Parallel-lane harness migration. |
| M72 | [status-M72.md](status-M72.md) | Structured API security alternatives and prompt-context v2. |
| M73 | [status-M73.md](status-M73.md) | Shared interview settlement and lifecycle-ranking boundary. |
| M74 | [status-M74.md](status-M74.md) | v0.2 consolidation and release-ready closure. |
| M75 | [status-M75.md](status-M75.md) | Browser workflow, authoring, evidence, and compatibility review closure. |
| M76 | [status-M76.md](status-M76.md) | Browser process, Docker, fallback, and registry regression closure. |
| M77 | [status-M77.md](status-M77.md) | Complete cross-package browser-authoring transaction contract and documentation. |
| M78 | [status-M78.md](status-M78.md) | Trusted browser-registration transaction v2, attestation, and executor handoff. |
| M79 | [status-M79.md](status-M79.md) | Complete locally: three full passes and review iteration 5; publication and operational adoption remain separate. |
| M80 | [status-M80.md](status-M80.md) | Complete locally: shared supervised authoring and packaging; W08 review iteration 2. |
| M81 | [status-M81.md](status-M81.md) | Complete locally: W8M integration, three complete qualification units and W09 review iteration 6. |
| M82 | [status-M82.md](status-M82.md) | Complete private v2 registration recovery attestation. |
| M83 | [status-M83.md](status-M83.md) | Complete private v3 registration recovery attestation. |
| M84 | [status-M84.md](status-M84.md) | Complete locally: UWS 1.11 and Browser 1.8/1.9 adoption; review iteration 1. |
| M85 | [status-M85.md](status-M85.md) | Complete locally: current-stack provider-free matrix and review iteration 1 pass. |
| M86 | [status-M86.md](status-M86.md) | Active: UWS 1.11 sandboxed real-browser qualification and current scenario evidence. |
| A01 | [status-A01.md](status-A01.md) | API-first browser-profile fallback authoring. |
| A02 | [status-A02.md](status-A02.md) | Additive browser authentication and named-session authoring. |
| A03 | [status-A03.md](status-A03.md) | Browsertools authoring handoff and guided-result consumption. |
| A04 | [status-A04.md](status-A04.md) | Explicit authenticated goal-directed Browsertools orchestration. |
| A05 | [status-A05.md](status-A05.md) | Live browser observation label, diagnostic, and negotiated-bound hardening. |
| A06 | [status-A06.md](status-A06.md) | Strict v2 MFA and typed dashboard output authoring. |
| A07 | [status-A07.md](status-A07.md) | Internal driver-agnostic iCoT authoring engine and shared approval writer. |
| A08 | [status-A08.md](status-A08.md) | Single-workspace loopback iCoT UI server and experimental JSON transport. |
| A09 | [status-A09.md](status-A09.md) | Enhanced Phase B transactional reliability, workspace drift protection, API v2, and polling status UX. |
| A10 | [status-A10.md](status-A10.md) | Interactive Phase C accessible authoring, review, conflict, and approval shell. |
| A11 | [status-A11.md](status-A11.md) | Real-browser Phase C qualification complete with mandatory sandboxed release evidence. |
| A12 | [status-A12.md](status-A12.md) | Authoring, provider, remote-source, and UI safety hardening. |
| A13 | [status-A13.md](status-A13.md) | iCoT UI answer, control, revision, recovery, and review remediation. |
| A14 | [status-A14.md](status-A14.md) | Browser authoring, synthesis, registry, and process-boundary hardening. |
| A15 | [status-A15.md](status-A15.md) | Engine-owned journey, API upload, and browser-capture acquisition lifecycle. |
| A16 | [status-A16.md](status-A16.md) | Unified API v3 UI, isolated bundled browser worker, package assessment, and handoff. |
| A17 | [status-A17.md](status-A17.md) | Acquisition, synthesis, and UI remediation closure. |
| A18 | [status-A18.md](status-A18.md) | Secret-safe browser registration intent, package, and dry-run handoff. |
| A19 | [status-A19.md](status-A19.md) | Complete private BAP, BCP, and BRP candidate lifecycle. |
| A20 | [status-A20.md](status-A20.md) | Complete unified browser transaction engine, API v4, UI, and terminal UX. |
| A21 | [status-A21.md](status-A21.md) | Guided iCoT browser-registration authoring v2. |
| A22 | [status-A22.md](status-A22.md) | Operator-authored UWS 1.9.1 content-trust declarations. |
| A23 | [status-A23.md](status-A23.md) | Exact Browsertools A10 consumer adoption and qualification. |
| A24 | [status-A24.md](status-A24.md) | Honest guided registration symbolic-slot reuse and deferred success proof. |
| A25 | [status-A25.md](status-A25.md) | Guided registration symbolic-binding false-positive closure. |
| A26 | [status-A26.md](status-A26.md) | One-attempt registration-authoring authority and terminal failure retention. |
| A27 | [status-A27.md](status-A27.md) | Registration terminal delivery and global containment propagation. |
| A28 | [status-A28.md](status-A28.md) | Private registration discovery inventory and independent owner review. |
| A29 | [status-A29.md](status-A29.md) | Typed registration 1.1 authoring and private input integration. |
| A30 | [status-A30.md](status-A30.md) | Reviewed verification authoring and registration 1.2 package binding. |
| P01 | [status-P01.md](status-P01.md) | Value-free browser verification package and review evidence. |
| P02 | [status-P02.md](status-P02.md) | Trusted execution and package-integrity v2 migration. |
| P03 | [status-P03.md](status-P03.md) | Value-free trusted browser execution through normal `openudon run`. |
| P04 | [status-P04.md](status-P04.md) | Immutable trusted execution and containment closure. |
| P05 | [status-P05.md](status-P05.md) | Complete prepare-only package construction, restrictive qualification, atomic promotion, and recovery. |
| P06 | [status-P06.md](status-P06.md) | Advisory content-trust quality and review evidence. |
| E01 | [status-E01.md](status-E01.md) | Browser authoring-to-handoff integration evaluation. |
| E02 | [status-E02.md](status-E02.md) | Authenticated authoring and trusted replay integrated qualification. |
| E03 | [status-E03.md](status-E03.md) | Exact authenticated authoring producer/consumer/replay seam qualification. |
| E04 | [status-E04.md](status-E04.md) | Deterministic real-browser loopback and opt-in public scenario suites. |
| E05 | [status-E05.md](status-E05.md) | Realistic guided-authoring browser journey release suite. |
| E06 | [status-E06.md](status-E06.md) | Honest, portable, deterministic, bounded evaluation evidence. |
| E07 | [status-E07.md](status-E07.md) | Authentication-real browser replay and exact compatibility evidence. |
| E08 | [status-E08.md](status-E08.md) | Unified UI real-Chromium acquisition-to-handoff qualification. |
| E09 | [status-E09.md](status-E09.md) | Typed, clean-root, sandboxed post-remediation release evidence closure. |
| E10 | [status-E10.md](status-E10.md) | Complete cross-package BxP transaction and release qualification. |
| E11 | [status-E11.md](status-E11.md) | Complete browser-registration runtime and BAP regression qualification. |
| E12 | [status-E12.md](status-E12.md) | UWS 1.9.1 content-trust compatibility qualification. |
| E14 | [status-E14.md](status-E14.md) | Registration foreground and authoritative private-checkpoint timing integration. |
| E15 | [status-E15.md](status-E15.md) | Verification failure evidence and complete registration 1.2 integration qualification. |
| E17 | [status-E17.md](status-E17.md) | Verification observability integration and exact revised driver qualification. |
| E16 | [status-E16.md](status-E16.md) | Native input identity for explicitly versioned incremental qualification consumers. |

## Candidate Directions

Candidates have no lane, ID, status file, or execution-order entry until a
fresh scope and dependency review promotes them.

| Direction | Why Deferred | Promotion Trigger |
|---|---|---|
| Further package/source-family integration | A03/P01/A04/E01/E02 own the approved Browsertools authoring/evidence integration; other API/event source metadata remains owned by apitools and public semantics by UWS. | Another upstream contract is published and an OpenUdon-owned package/review outcome beyond this sequence is explicitly scoped. |
| Automated real-provider release evidence | Provider runs spend quota and can produce sensitive output; current policy remains local/manual. | Protected credentials, redaction, retention, spend bounds, and review-required CI policy are approved. |
| Trusted-runner capability expansion | OpenUdon hands approved packages to an external executor and must not absorb runtime semantics. | A public handoff/evidence gap is demonstrated without importing private runtime behavior or weakening approval gates. |
| UI-owned LLM drafting | The primary UI now owns deterministic acquisition, interview, review, and handoff, while extractor drafting and repair remain terminal/external-orchestration functions. | A separately reviewed engine mutation can invoke an optional extractor under exact-revision protection, persist every proposal as confirmation-required state, and preserve the no-silent-acceptance boundary. |

## Notes

- Historical full real-LLM smoke baseline in README: 2026-04-28, `gemini-2.5-flash`, structured
  output path, ten original examples passed with zero legacy extraction fallbacks. Current local
  real-LLM defaults use `copilot-api` with `gpt-5.4-mini`.
- M35-M37 retain historical local release-candidate and product-smoke evidence.
  No corresponding public GitHub tag or release exists; M69 owns the first
  public tag as v0.1.0. Real provider outputs remain ignored and local/manual.
- Advisory n8n reducibility fixtures remain part of the eval corpus. They should not introduce
  n8n-specific runtime behavior into OpenUdon or udon; use explicit intent, OpenAPI, and generic
  `fnct` or control-flow modeling.
- Normal deterministic gates remain `go test ./...`, `go vet ./...`, `make check`, and
  `git diff --check`. GitHub Actions runs provider-free public module gates and the repository
  boundary guard with `GOWORK=off`; sibling readiness, `check-doc-memory`, and real-provider release
  evidence remain local/manual.
- `openudon run` is the only OpenUdon-owned path that invokes a trusted executor runner, and it
  requires approval JSON plus a valid handoff package.
- OpenUdon no longer imports udon as a Go module; udon is an optional external trusted executor
  behind the run-config handoff.
- After a major review or milestone, check whether [tabilet/evolution/](../evolution/) needs a new
  prompt/result version.

## Milestones

### 1. Post-POC Baseline

- Preserve the known-good eval and synthesis baseline.
- Keep deterministic quality gates covering project policy, OpenAPI availability, intent validity,
  workflow compilation, expected-plan matching, UWS validation, review evidence, and secret scanning.
- Document deterministic checks versus optional real-provider smoke tests.

Acceptance: committed docs and eval reports describe current behavior without overstating public API
stability or release readiness.

### 2. Eval Corpus And Reference Discipline

- Expand curated eval briefs across OpenAPI auth schemes, pagination, request bodies, response
  extraction, writes, multi-service chains, runtime-only functions, approved/denied runtimes, and
  negative policy cases.
- Keep advisory n8n reducibility fixtures for Airtable, Gmail, Google Drive, HubSpot, Jira,
  OpenWeatherMap, PagerDuty, Slack, and Trello as evidence that scanner/OpenAPI-backed workflows
  can be expressed as OpenUdon intent without adding n8n-specific runtime behavior.
- Classify reference issues as advisory, warning, or blocking.
- Keep per-fixture reference policies and triage notes.
- Gate release evals on pass rate, structured mode, attempts, blocking reference issues, and
  secret-scan failures.

Acceptance: eval reports identify behavioral regressions separately from acceptable naming or
review-text drift.

### 3. Structured Output And Provider Drift

- Use provider-native structured generation where supported and keep legacy extraction as fallback.
- Report structured fallback count, provider failures, model availability, attempts-to-pass, release
  gate failures, and comparison deltas.
- Keep real-provider evals local/manual until protected secret and redaction automation exists.

Acceptance: release evidence can distinguish deterministic OpenUdon regressions from provider drift.

### 4. Quality Gate Hardening

- Tighten data-flow checks for missing dependencies, ambiguous sources, invalid response paths, and
  undeclared function inputs.
- Harden credential checks around binding declarations, OpenAPI security schemes, request placement,
  and secret-value leakage.
- Expand side-effect checks for write operations, customer communications, command/SSH runtimes, and
  production endpoint language.
- Require review evidence for side effects, unresolved risks, skipped execution, credential binding
  names, approval states, sandbox proof runs, and trusted-runner handoff.

Acceptance: common artifact mistakes fail with precise quality codes and concrete repair guidance.

### 5. Workflow Artifact Power

- Preserve richer UWS-compatible workflow patterns through intent, workflow HCL, UWS export, plan,
  review, and quality checks.
- Cover switch, loop, structural results, success criteria, failure actions, retries, explicit
  timeout metadata, and workflow idempotency metadata where public sibling contracts exist.
- Keep prompt defaults constrained to explicit project/intent requests for retries, timeouts, and
  idempotency.

Acceptance: OpenUdon can validate richer workflow artifacts without moving public semantics or runtime
execution behavior into OpenUdon.

### 6. iCoT Authoring

- Guide operators from broad project ideas to `project.md` and `workflows/intent.hcl`.
- Support optional LLM kickoff/refine/disambiguate roles while keeping offline manual authoring.
- Autosave incomplete sessions, save transcripts, support reconcile/lint/replay, and write final
  artifacts atomically.
- Improve OpenAPI operation ranking, request mapping inference, readiness checks, grouped
  questions, and confidence/evidence classification.

Acceptance: a trusted user can author a reviewable OpenUdon workflow package without reading
implementation code.

### 7. Safety And Trusted Execution

- Define the minimum review package for trusted execution.
- Emit `expected/review-handoff.json` using the stable `apitools.review-handoff.v1` wire
  version, with OpenUdon-owned validation and lifecycle behavior.
- Generate approval JSON from the current package digest.
- Validate handoff manifest, stored/current quality, approval scope, expiry, digest, and tier/state
  compatibility before udon invocation.
- Keep synthesis, build, promote, assess, iCoT, and eval free of production side effects.

Acceptance: side-effectful execution happens only through approved local trusted-runner gates.

### 8. Local Checks And Release Process

- Keep `go test ./...`, `go vet ./...`, `make check`, and `git diff --check` as normal
  deterministic gates.
- Keep `make release-check` as deterministic pre-tag release readiness.
- Keep `make release-eval` separate as opt-in real-provider release evidence with expanded-corpus
  minimum brief count.
- Keep release notes recording model, prompt version, corpus size, pass rate, comparison baseline,
  provider drift, and known gaps.

Acceptance: routine development and deterministic release readiness stay fast and provider-free,
while release confidence can include separately recorded manual provider evidence.

### 9. Product Usability

- Keep CLI help, operator checklist, onboarding, project template, eval gallery, and quality repair
  hints aligned with current behavior.
- Keep README as the concise operator entrypoint and memory-bank as project source of truth.

Acceptance: a new trusted operator can author, synthesize, assess, evaluate, and prepare a handoff
from documented commands.

### 10. Cross-Repo Dependency Stewardship

- Track UWS semantics, udon lowering/runtime compatibility, review approval handoff, provider
  drift, private checkout readiness, runtime/profile evals, and expanded release evidence.
- Close OpenUdon-owned slices with regression coverage and open sibling work only when a reusable
  upstream gap is proven.

Acceptance: OpenUdon remains thin and does not absorb sibling ownership.

### 11. Public OpenUdon Package Boundary

- Complete the OpenUdon lifecycle migration. Status: done. OpenUdon now owns iCoT/progressive loop,
  prompt transcript/replay, JSON completion fallback, artifact sets, review handoff validation,
  package digest, symbolic binding contracts, credential scanning, and review metadata locally.
- Preserve the final `../apitools` keep boundary: OpenAPI-first API metadata search, discovery,
  import/lowering, download, local scanning, validation, operation indexing, operation summaries,
  auth/security summaries, catalog metadata, operation ranking, CLI search/import, and cache support.
  Status: done.
- Remove or move downstream all non-metadata `../apitools` lifecycle APIs: generic authoring
  structs/flows, iCoT loop/session/transcript helpers, JSON completion fallback, review
  handoff/state machine, package digest, credential scans, binding contracts, leaf adapter/review
  package helpers, LLM provider helpers, and Context7/documentation authoring context. Status: done.
- The IaC sibling is parked and is not a compatibility gate for this narrowing. It must move any
  lifecycle dependencies into its own packages before it resumes tracking current apitools.
- Migrate `../udon` before deleting APIs. Udon must move runtime-plan leaf/review helpers and
  `apitools/llm` usage into udon-owned code and keep only API metadata search/import/lowering/index
  usage.
  Status: done.
- Keep OpenUdon on local `internal/authoring` lifecycle helpers and add or keep a static guard that
  OpenUdon production packages do not import non-metadata `apitools` lifecycle APIs. Status: done.
- Hard-narrow `../apitools` only after downstream consumers compile without those APIs: delete or
  move non-metadata lifecycle packages/files, rewrite README/docs around the OpenAPI-first API
  metadata boundary, and keep authoring/review/handoff material only as historical migration notes
  when needed. Status: done.
- Keep `apitools.review-handoff.v1` only as a wire compatibility string while downstream artifacts
  still need it, not as active `../apitools` lifecycle ownership.
- Split OpenUdon's remaining udon executor integration into a trusted executor handoff based on UWS Document, staged API source files, non-secret run config, and runtime credential resolution. Status: done; OpenUdon stages reviewed artifacts into a fresh executor-visible directory under the run workdir and invokes udon through canonical `OPENUDON_EXECUTOR` as either an absolute CLI path or `docker://<image>`.
- Harden the trusted executor handoff so every staged OpenAPI file is digest-covered, symlinked
  OpenAPI artifacts are rejected, Docker receives only declared credential env names, and bad
  OpenAPI operation IDs fail generation instead of producing partial request maps. Status: done.

Acceptance: OpenUdon owns lifecycle APIs locally, `../apitools` contains only API metadata tooling,
`../udon` compiles without non-metadata apitools lifecycle APIs, the IaC sibling is explicitly parked, static guards
prevent regression, and OpenUdon's public build no longer relies on broad shared apitools product workflow APIs or udon Go packages.

Verification plan:

- OpenUdon: `go test ./...`, `go vet ./...`, `make check`, and `git diff --check`.
- `../apitools`: `go test ./...`, `go vet ./...`, `git diff --check`, plus CLI smoke coverage for
  `search` and `import`.
- Downstreams: `(cd ../udon && go test ./...)`. The IaC sibling is parked and not a gate.
- Static guards: `rg` for removed non-metadata lifecycle symbols in OpenUdon, udon, and apitools docs.

### 12. Package Artifact And Local OpenAPI Safety Hardening

- Harden OpenUdon package artifact validation and digest inputs for all required handoff files. Status:
  done; required OpenUdon handoff inputs now share safe relative path validation, manifest inventory
  checks, package-root and regular-file validation, digest input validation, and trusted-runner
  staging guards.
- Harden `../apitools` local OpenAPI reads with symlink, type, and size checks. Status: done;
  `LocalFiles`, `BuildOperationInventory`, and `LoadOperationIndex` now share bounded local file
  reads that reject symlinked roots/paths/parents, directories, special files, and oversized
  path-backed documents.
- Replace the trusted executor shell/Python runner with Go run-config parsing and staging. Status:
  done; `internal/udonrunner` and `cmd/udon-runner` now validate config JSON, stage
  workflow/OpenAPI inputs, and exec the configured binary or Docker executor by argv.
- Split `workflowintent` into intent model/HCL, provider clients, and OpenAPI adapter modules.
  Status: done; the package name and exported API remain unchanged while the deleted monolith is
  replaced by focused `intent.go`, `provider_client.go`, `openapi.go`, and shared helpers.

Acceptance: package roots and required handoff inputs cannot be symlinks, directories, special
files, unsafe relative paths, or digest/staging bypasses; sibling-owned hardening remains tracked
without moving ownership into OpenUdon; the review-follow-up hardening group is closed.

### 15. Agentic UWS Authoring For Common SaaS Workflows

Make AI-assisted UWS authoring for common OpenAPI-backed workflows the next
primary OpenUdon product slice. Status lives in
[status-M15.md](status-M15.md) because the milestone has multiple
implementation tasks.

Goal:

- Let developer/operator users describe useful SaaS workflows in natural
  language or guided CLI prompts and receive normal OpenUdon/UWS review
  packages.
- Use AI where it helps: goal clarification, operation selection, request and
  response mapping drafts, and unresolved-assumption explanation.
- Keep everything after UWS/OpenUdon artifact generation deterministic:
  validation, review evidence, package digest, approval, and trusted executor
  handoff.
- Use n8n and `../try-n8n` as service-priority and workflow-pattern evidence,
  not as the central product interface or runtime dependency.

Scope:

- Improve `cmd/icot` and `openudon synthesize/build/assess` around common
  OpenAPI-backed workflow authoring.
- Start with service workflows identified by n8n/try-n8n evidence and existing
  OpenUdon advisory fixtures: Slack message post, Gmail message send, Jira issue
  get/create, HubSpot list, Google Drive upload, Airtable record lookup,
  PagerDuty user lookup, Trello list lookup, and OpenWeatherMap current weather.
- Keep OpenAPI documents, operation IDs, credential binding names, request
  mappings, data-flow expectations, review evidence, and quality checks crisp
  for those services.
- Use n8n fixture metadata and `../try-n8n` scanner output as corpus evidence
  for which services and operation shapes matter.
- Emit diagnostics and review TODOs for unsupported nodes, expressions,
  binary-data behavior, item batching, custom code, triggers that need runtime
  semantics, and ambiguous OpenAPI mappings when n8n-derived examples are used.
- Keep credential values out of generated artifacts; preserve only symbolic
  binding names.

Non-goals:

- Do not make n8n import the primary M15 deliverable.
- Do not execute n8n workflows, import n8n runtime internals, emulate full n8n
  item semantics, or add n8n-specific workflow semantics to UWS.
- Do not resume broad desired-state provider conversion in this milestone.

Acceptance: a new user can use OpenUdon's AI/CLI authoring flow to produce
reviewable UWS/OpenAPI packages for several common SaaS workflows, with clear
symbolic credentials, request/data-flow mappings, quality reports, review
evidence, and trusted-handoff posture. n8n-derived evidence informs service
priority and examples, while detailed n8n workflow import is deferred to a
later milestone. Status: initial M15 planning and public authoring contract are
done; follow-on hardening continues in M16.

### 16. SaaS Authoring Corpus Hardening

Turn the M15 service-priority list into a reliable regression corpus. Status
lives in [status-M16.md](status-M16.md) because this milestone has multiple
implementation tasks.

Goal:

- Make Slack, Gmail, Jira, HubSpot, Google Drive, Airtable, PagerDuty, Trello,
  and OpenWeatherMap useful as concrete authoring references rather than only
  planning examples.
- Decide which fixtures are strict OpenUdon-native golden coverage, which are
  advisory n8n-derived evidence, and what is required for an advisory fixture
  to graduate.
- Normalize the starter OpenAPI slices so operation IDs, required request
  fields, response fields, security schemes, and credential binding names are
  visible to prompts and deterministic quality checks.

Scope:

- Audit the existing native and n8n-derived fixtures for the selected services.
- Fill small OpenAPI-schema gaps when the local fixture already owns the slice.
- Add or update `reference/authoring.json` metadata for native SaaS authoring
  fixtures.
- Align project briefs, reference intents, fixture policies, docs, and quality
  expectations around symbolic credential bindings and request/response
  mappings.
- Keep n8n provenance in `reference/n8n.json` for advisory fixtures, but do not
  add n8n runtime behavior or import semantics.

Acceptance: the selected service corpus has an auditable readiness matrix,
stable fixture policy, clear credential-binding conventions, and at least a
small strict OpenUdon-native SaaS golden set that can detect operation,
request-mapping, data-flow, and side-effect regressions. Status: done; the
initial strict native set is Slack message audit log, Gmail send audit receipt,
and Slack-to-Jira issue intake, with the rest of the selected SaaS services
kept as advisory or native-pattern evidence until graduation.

### 17. Guided SaaS Authoring UX

Improve the practical authoring loop after the corpus is stable.

Goal:

- Let a new user describe a SaaS workflow and get a high-quality `project.md`
  plus `workflows/intent.hcl` with fewer manual repairs.
- Make `cmd/icot`, synthesis prompts, readiness feedback, and lint output
  ask crisp questions about service, operation, inputs, outputs, credentials,
  side effects, and fallback behavior.

Scope:

- Improve guided prompts and autosave/resume behavior only where it reduces
  repeated authoring friction.
- Use the M16 corpus as prompt/eval evidence.
- Prefer deterministic readiness and repair messages over broad LLM retries.

Acceptance: common single-service SaaS briefs converge to reviewable artifacts
with clear unresolved assumptions and concrete repair guidance. Status: done;
guided iCoT readiness now asks for listed operation IDs, symbolic credential
bindings, request field sources, known output sources, and explicit
read-only/sandbox/approval posture.

### 18. Multi-Service SaaS Workflow Patterns

Expand from single-service authoring to common deterministic SaaS chains.

Goal:

- Cover repeatable workflow shapes such as lookup-then-notify,
  ticket-then-message, send-then-audit, ticket-alert-archive,
  paginate-then-summarize, and webhook-then-create/update.

Scope:

- Add or harden multi-service fixtures using selected services from M16.
- Keep data-flow evidence explicit across service boundaries.
- Keep transformations in declared `fnct` steps unless public UWS/udon
  semantics support a more generic runtime.

Acceptance: OpenUdon can author, build, assess, and review representative
multi-service SaaS packages with auditable cross-step bindings and side-effect
policy. Status: done; the strict multi-service set is Slack-to-Jira issue
intake, incident response archive, and order fulfillment chain, with pagination
and webhook send retained as supporting pattern coverage.

### 19. SaaS Review And Trusted-Handoff Confidence

Strengthen the review and handoff posture for side-effectful SaaS workflows.

Goal:

- Make approval evidence, credential binding posture, sandbox/production
  boundaries, package digests, and trusted-runner handoff clear enough for real
  operator review.

Scope:

- Improve review evidence for SaaS side effects and credential scopes.
- Add stronger fixture coverage for sandbox-only and production-handoff
  language.
- Keep execution behind `openudon run` and trusted executor approval checks.

Acceptance: side-effectful SaaS workflows produce review packages whose risks,
credential bindings, approval state, digest, and trusted-runner command are
easy to audit before any executor receives a run config. Status: done; generated
review evidence now includes side-effect risk, credential scope, approval JSON,
dry-run, package digest, and trusted-runner run-config boundary sections gated
by deterministic quality checks.

### 20. End-To-End SaaS Authoring Trials

Use realistic user-facing SaaS briefs to measure the complete authoring path
after M15-M19.

Goal:

- Prove whether a new trusted user can start from a normal SaaS workflow request
  and reach a reviewable OpenUdon package without maintainer-only knowledge.
- Capture where `cmd/icot`, synthesis, readiness checks, fixture OpenAPI
  coverage, docs, or repair hints still force manual intervention.

Scope:

- Select 5-8 realistic briefs across the strict and advisory service families:
  Slack, Gmail, Jira, Google Drive, HubSpot, Airtable, PagerDuty, Trello,
  OpenWeatherMap, and one synthetic internal API chain.
- Run each brief through guided authoring, build/assess, review evidence, and
  approval-template/dry-run gates when the package is side-effectful.
- Record trial evidence as ignored local run artifacts first, then promote only
  stable patterns, docs, fixtures, or tests.
- Keep live-provider execution optional/manual; deterministic package evidence
  remains the acceptance target.

Acceptance: OpenUdon has a documented trial matrix showing which SaaS briefs
converge, which need repair, which gaps are fixture/OpenAPI coverage issues, and
which product improvements should be implemented before broadening the corpus.
Status: done; M20 documented eight trials, fixed one Slack fixture contract
wording issue, confirmed three package/dry-run paths, and identified
security/request-field normalization as the main M21 prerequisite.

### 21. Strict SaaS Corpus Expansion

Promote the strongest trial-backed SaaS workflows into strict regression
coverage.

Goal:

- Expand from the small strict SaaS set into a broader but still maintainable
  corpus that proves common request mapping, response extraction, credentials,
  side effects, and review-handoff behavior.

Scope:

- Promote only trial-backed workflows whose project brief, local OpenAPI slice,
  reference intent, plan, authoring metadata, policy, and review posture are
  deterministic.
- Prefer service diversity over volume: HubSpot, Google Drive, Airtable,
  PagerDuty, Trello, OpenWeatherMap, and one additional multi-service SaaS chain
  should be considered before adding duplicate Slack/Jira examples.
- Add or harden `reference/authoring.json`, fixture policy metadata, eval
  gallery entries, and corpus docs for each promotion.
- Keep n8n-derived examples advisory unless they satisfy the OpenUdon-native
  graduation criteria.

Acceptance: the strict SaaS corpus covers a wider set of service families and
workflow shapes while remaining provider-free, deterministic, and release-gated.
Status: done; bearer security request placement is normalized, `weather-toronto`
has strict authoring metadata, the strict single-service and multi-service SaaS
fixtures build from reference intents, and the promoted set passes sandbox
trusted-runner dry-run gates without provider credentials.

### 22. n8n Pattern Bridge

Build a narrow, review-first bridge from n8n evidence to OpenUdon authoring
without importing n8n runtime behavior.

Goal:

- Make n8n useful as a discovery and prioritization source for OpenUdon users
  who know a workflow shape but want deterministic UWS/OpenAPI artifacts.

Scope:

- Define a small n8n pattern summary format that records nodes, services,
  operations, credentials by symbolic name, data-flow hints, unsupported
  semantics, and OpenAPI mapping candidates.
- Add a local command or documented harness only if it can emit review evidence
  without pretending to execute or faithfully import n8n workflows.
- Preserve unsupported nodes, expressions, binary data, item batching, custom
  code, triggers, and scheduling as diagnostics or TODO evidence.
- Keep the output as project-authoring assistance: `project.md` and
  `workflows/intent.hcl` candidates still go through normal OpenUdon validation,
  review, package, approval, and trusted-handoff gates.

Acceptance: a small set of n8n-inspired examples can be summarized into
OpenUdon authoring inputs with explicit unsupported-semantics diagnostics and no
n8n runtime dependency.
Status: done; `openudon.n8n-pattern-summary.v1` summaries cover Slack, Google
Drive, and HubSpot advisory fixtures, `openudon n8n-bridge validate` checks the
contract deterministically, unsupported n8n semantics are documented as
diagnostics or TODO/manual contracts, and bridge output remains authoring
assistance only.

### 23. SaaS Operator Release Readiness

Package the SaaS authoring story into a release-quality operator path.

Goal:

- Make the SaaS slice understandable and repeatable for a new operator:
  author, build, assess, review, approve, dry-run, and archive evidence.

Scope:

- Add or update operator docs, tutorials, release checklist entries, and demo
  commands around the selected strict SaaS examples.
- Ensure public docs explain what OpenUdon does and does not do versus n8n,
  live providers, desired-state engines, external orchestration, and udon.
- Tighten release evidence around deterministic SaaS fixtures, optional
  real-provider evals, provider drift, and trusted-runner dry-run output.
- Do not broaden live-provider automation or production execution policy.

Acceptance: a release candidate can demonstrate the SaaS authoring workflow
from brief to trusted-runner dry run with deterministic evidence and clear
non-goals.
Status: done; the SaaS operator release path documents provider-free demo
commands for `gmail-send-audit-receipt` and `order-fulfillment-chain`, release
stewardship and release-note templates capture deterministic and optional
provider-drift evidence, boundary docs clarify OpenUdon versus n8n/live
providers/desired-state engines/external orchestration/udon, and both selected demos pass sandbox
trusted-runner dry-run smoke from ignored workdirs.

### 24. Release Gate Consolidation And Evidence Automation

Consolidate the post-M23 SaaS release checklist into a single local maintainer
gate while preserving the existing fast deterministic pre-tag gate.

Goal:

- Let maintainers run one provider-free command that proves the documented SaaS
  release evidence path: deterministic gates, documentation checks, bridge
  validation, selected fixture lint, and trusted-runner dry-run demos.

Scope:

- Keep `make release-check` as the fast deterministic pre-tag gate.
- Add a comprehensive local `make release-saas-check` target for SaaS release
  evidence.
- Keep the target provider-free and local-only: no real LLM calls, live SaaS
  providers, n8n execution, desired-state engine execution, orchestration service
  dependency, production effects, or trusted executor invocation.
- Update release docs, release-note evidence, and memory-bank status so the
  command is the canonical M24 gate.

Acceptance: a maintainer can run `make release-saas-check` from a normal local
checkout and receive provider-free SaaS release evidence covering release-check,
UWS validation, doc-memory, n8n bridge validation, strict MkDocs build, selected
strict fixture lint, and Gmail/order-fulfillment sandbox dry-run demos under
ignored `.openudon-run/...` output.
Status: done; `make release-saas-check` is the comprehensive local SaaS release
gate, release docs and release-note evidence point to the target, and the gate
has passed with provider-free fixture lint plus Gmail and order fulfillment
trusted-runner dry-run demos.

### 25. First-Class Provider Catalog CLI Integration

Expose the upstream `../apitools` first-class provider catalog through thin
OpenUdon authoring commands.

Goal:

- Let operators inspect provider-owned OpenAPI, Discovery, Smithy, Stone,
  human-docs, and security-overlay metadata from OpenUdon before falling back
  to public search or hand-written OpenAPI slices.

Scope:

- Add `openudon catalog list`, `inspect`, `advisory`, `specs`,
  `security-report`, and `import-openapi`.
- Keep catalog data advisory; explicit local `openapi/` files and user-provided
  OpenAPI inputs remain authoritative package inputs.
- Permit `import-openapi` only for actual catalog OpenAPI references. Discovery,
  Smithy, Stone, and human-docs references must remain advisory until a user
  supplies or generates a valid OpenAPI document.
- Keep catalog maintainer workflows such as refresh, refresh-report, stats, and
  security-audit in `../apitools` rather than exposing them as normal OpenUdon
  operator commands.
- Update operator docs and tests for the command surface.

Acceptance: operators can inspect first-class provider catalog metadata and
import provider-owned OpenAPI refs into package-local `openapi/` directories
without executing API operations, resolving credentials, changing workflow
semantics, or treating native Discovery/Smithy metadata as OpenAPI.
Status: done; `openudon catalog` exposes provider list, inspect, advisory,
specs, security-report, and direct OpenAPI import commands. Direct import is
restricted to actual catalog OpenAPI refs; native Discovery, Smithy, Stone, and
human-docs entries remain advisory metadata. See [status-M25.md](status-M25.md).

### 26. iCoT Catalog Planning And Operation-Detail Mapping Pipeline

Harden the iCoT SaaS authoring pipeline after first-class provider catalog
inspection by adding bounded LLM assistance at the two places where it reduces
operator friction without weakening local validation.

Goal:

- Let iCoT use deterministic catalog hints plus a compact LLM catalog plan to
  select relevant local catalog artifacts immediately after the workflow goal.
- Keep local validation authoritative: selected provider/artifact tuples must
  match the deterministic shortlist before any migration, and early plan output
  may seed only provider/capability-level steps.
- Let iCoT use selected operation details to draft required request field
  mappings before asking the operator to map fields manually.
- Let iCoT run one advisory pre-final flow review to catch cross-step data-flow
  mistakes that deterministic request/schema checks may miss.
- Preserve the existing operation-detail draft loop, final confirmation gate,
  transcript evidence, and deterministic validation before saving
  `intent.hcl`.

Scope:

- Add an extractor `CatalogPlan` method and a compact `catalog_plan` prompt
  that excludes full OpenAPI, Google Discovery, advisory overlay, schema,
  request field, response path, credential, and secret content.
- Validate all early catalog selections against `BuildCatalogHints` and
  `CatalogMigrationCandidates`; reject unknown providers, unknown artifact
  keys, invented paths, non-migratable artifacts, and unsafe step details.
- Migrate selected validated artifacts and automatically include advisory
  overlays for selected providers when available.
- Materialize matching apitools security overlay sidecars beside migrated
  OpenAPI, Google Discovery, or AWS Smithy sources when exact catalog metadata
  supports the match; unmatched overlays stay non-blocking retrieval notes.
- Record `catalog_plan_call`, `catalog_plan_result`, and
  `catalog_plan_rejected` transcript events.
- Add a focused LLM draft opportunity when readiness reports missing required
  request mappings after operation selection, then fall back to the existing
  manual field-mapping prompt if the LLM cannot produce safe mappings.
- Add a focused pre-final `ReviewDraft` pass that returns structured warnings
  only, does not mutate the draft by default, and records transcript evidence.
- Keep concrete operation IDs, request mappings, credentials, response paths,
  and final intent details sourced from local operation metadata plus final
  operator confirmation.
- Park `--review-repair` in M26 until repair iterations can apply deterministic
  or operator-approved changes; M28 later graduates it as experimental opt-in
  behavior.

Acceptance: a catalog-backed SaaS brief such as "get weather of Toronto,
Canada, and then send the report using Google Gmail" can use cached
OpenWeatherMap advisory OpenAPI and Gmail Discovery artifacts, ask for listed
operation IDs, use local operation details to insert legal prework steps such
as OpenWeatherMap geocoding before weather lookup, draft remaining request
mappings, materialize matching security overlay sidecars, and still reject
malformed or invented catalog-plan output before saving a reviewable
`intent.hcl`. Status: in progress; see [status-M26.md](status-M26.md).

### 28. iCoT Decision Evidence And Repair-Loop Evaluation

Improve iCoT as an interactive plan-and-execute authoring loop without turning it
into an unbounded autonomous agent.

Goal:

- Preserve compact, explicit decision evidence for catalog artifact selection,
  operation selection, request mapping, output selection, side-effect scope, and
  pre-final flow review.
- Thread that evidence into downstream LLM prompts as user-visible rationale,
  not hidden chain-of-thought.
- Evaluate experimental `--review-repair` behavior that can apply one or two
  bounded repair attempts only when each iteration makes a deterministic or
  operator-approved draft change.
- Use confidence and review severity to decide when `--prompt-mode fast` may
  auto-accept defaults, when `normal` should show but not block on defaults,
  and when all modes should ask the operator.
- Treat ambiguous side-effect commitment, including provider-as-verb phrasing,
  as understanding-only evidence until the operator explicitly confirms the
  provider action.
- Add a small cross-automation sample corpus inspired by common n8n, Zapier,
  Make, Pipedream, and Tray-style workflows while keeping OpenUdon-native
  intent, API source metadata, review evidence, and trusted-runner boundaries.

Scope:

- Add transcript/session fields for short decision evidence and confidence at
  the catalog, operation, mapping, output, side-effect, and review stages.
- Feed prior decision evidence into later prompts so downstream stages can
  challenge plausible but wrong earlier choices.
- Add deterministic policy for confidence-driven prompting and auto-acceptance.
- Add deterministic policy that side-effectful operation commitment needs
  explicit provider/action wording or a forced operator answer, including in
  `fast` mode.
- Prototype review repair behind an explicit flag or internal experiment path;
  do not make repair loops default until eval fixtures show measurable benefit.
- Keep bounded halt conditions and fall back to the operator after failed repair
  attempts instead of repeatedly reviewing an unchanged draft.
- Select a compact sample set, roughly six to eight workflows, covering common
  automation patterns:
  lead capture to CRM plus team notification; support intake to ticket plus
  status message; form or webhook to spreadsheet/Airtable plus confirmation
  email; file upload to Drive archive plus notification; weather or external
  enrichment to email/report delivery; order or invoice event to inventory,
  fulfillment, or accounting follow-up; incident alert to PagerDuty/Jira/Slack;
  and a negative or ambiguous-source case that should force user confirmation.
- Treat third-party automation galleries as service/action vocabulary and
  prioritization evidence only. Do not import n8n, Zapier, Make, Pipedream, or
  Tray runtime semantics into OpenUdon or UWS.

Acceptance: iCoT records compact decision evidence, uses confidence to control
prompt volume, evaluates bounded review repair without making it default, and
measures realistic cross-automation fixtures with strict warning/review limits.
Status: implemented; see [status-M28.md](status-M28.md).

### 29. iCoT Flow Remediation And Draft Expansion

Make iCoT better at turning incomplete natural-language goals into the best
defensible draft without pretending every brief is complete.

Goal:

- Preserve the principle that iCoT should build as much as local metadata and
  the user's goal can justify, then stop exactly where user intent is genuinely
  underspecified.
- Classify pre-final flow-review gaps into general remediation categories such
  as missing transform/report step, missing API prework step, disconnected
  notification/message, ambiguous output, incompatible operation, unavailable
  source/artifact, and unclear user intent.
- Add a bounded remediation-planning stage that may propose safe structural
  changes, especially local `fnct` transform/report steps, without inventing
  provider operations, API artifacts, credentials, or side-effect policy.
- Ask precise forced follow-up questions when the next draft change depends on
  user intent rather than local evidence; `fast` may skip defaults but must not
  hide true ambiguity.
- Stop before committing side-effectful operations from confusing
  provider-as-verb workflow goals such as "gmail the report", "slack Bob", or
  "jira the bug".
- Keep unresolved issues as visible non-executable `intent.hcl` comments when
  iCoT cannot safely repair or ask a sufficiently precise question.

Scope:

- Extend flow-review handling with a gap classifier and remediation action model
  separate from the narrow `--review-repair` mapper.
- Allow safe proposed `fnct` steps only when the goal clearly asks for produced
  content and an existing prior step provides the needed input.
- Allow API prework only for operations already listed in local metadata with
  details available; otherwise ask the operator or preserve a warning.
- Keep operation changes, credential choices, side-effect policy, and new API
  source selection outside automatic repair.
- Use the M28 fixture set plus additional general workflow variants to prove
  behavior across report delivery, notification, file upload, enrichment,
  order/fulfillment, incident, and negative ambiguous-source cases.

Acceptance: when pre-final review or strict side-effect commitment policy finds
a gap outside the narrow repair surface, iCoT either proposes a bounded
structural remediation from known local context, asks a precise
user-refinement question, or preserves a precise comment-only warning. The
solution must be workflow-pattern general and not special-cased to the
weather/Gmail or Gmail audit examples. Status: implemented; see
[status-M29.md](status-M29.md).

### 30. Build From Intent Hardening

Make `openudon build` the reliable deterministic path from reviewed
`workflows/intent.hcl` to `workflow.hcl`, UWS YAML, expected plan, review,
handoff, refinement, and quality artifacts. M30 is build-only: it does not
change iCoT prompt behavior, `openudon synthesize` intent generation, or
trusted execution.

Goal:

- Treat a user- or iCoT-authored `intent.hcl` as the source of truth and avoid
  silent operation or provider inference during build.
- Support all first-class package-local API source directories consistently:
  `openapi/`, `google-discovery/`, `aws-smithy/`, and legacy-readable
  `discovery/`.
- Keep build deterministic and review-first, with precise quality failures or
  warnings when local metadata cannot prove an operation, required parameter,
  credential binding, or response path.
- Preserve the separation between artifact generation and execution; build must
  never perform side effects or require approval credentials.

Scope:

- Completed first repair: build now indexes package-local Google Discovery and
  AWS Smithy operations for expected-plan and quality checks, accepts native
  operation ID aliases such as dotted Google Discovery IDs and normalized
  underscore IDs, accepts `credentials.<binding>` sources named by project
  credential policy, and downgrades unresolved array `$ref` response paths from
  hard missing to opaque review warnings.
- Completed Discovery request-body repair: Google Discovery request schemas now
  expose required body fields such as Gmail `raw` to iCoT review repair, build
  request mapping, expected-plan generation, and quality checks. Delivery APIs
  that require rendered message/body content should use an explicit local
  `fnct` render step before the side-effecting API step.
- Completed advisory security sidecar repair: build reads package-local
  `*.security.{json,yaml}` and `*.security-overlay.{json,yaml}` files next to
  their associated OpenAPI, Google Discovery, or AWS Smithy source and uses
  them as review/quality evidence for credential requirements without treating
  the sidecars as API source contracts. Associated sidecars are included in
  handoff inputs and package digests when present. iCoT catalog migration may
  place those sidecars during authoring, while build remains offline and never
  retrieves them.
- Extend native-source metadata parity where local metadata supports it:
  required request parameters, intrinsic security/credential requirements, and
  response path checks for Google Discovery and AWS Smithy should be checked
  with the same quality-report posture as OpenAPI instead of being skipped or
  treated as OpenAPI.
- Remove any unnecessary LLM-provider dependency from the build-only path. A
  reviewed intent should build offline; provider/model values may remain review
  labels, but build should not need a chat client to regenerate deterministic
  artifacts.
- Add a focused build sample matrix or target that runs representative
  intent-authored packages, including OpenAPI, Google Discovery, AWS Smithy,
  multi-source, no-source local function, and negative invalid-operation cases.
- Keep generated review, handoff, plan, and quality artifacts stable when only
  build-owned metadata evidence changes.

Acceptance: `openudon build --example <pkg>` succeeds or fails deterministically
from `intent.hcl` alone, validates typed API sources without OpenAPI-only blind
spots, reports opaque metadata as warnings rather than fabricated certainty, and
has a focused regression target for representative build packages. Status:
implemented; see [status-M30.md](status-M30.md).

### 31. Trusted Execution Hardening

Make `openudon run` a stronger, auditable trusted-handoff gate after M30 build completion. M31 stays
inside OpenUdon's wrapper boundary: it hardens approval and run-config parsing, dry-run staging,
package digest enforcement, executor handoff evidence, and sandbox/production boundary checks
without adding UWS semantics or implementing executor/provider behavior.

Goal:

- Ensure approval JSON and `openudon.executor-run.v1` inputs fail closed on malformed, trailing, or
  unknown JSON fields, unsafe paths, missing package inventory, invalid package digests, invalid tier
  state, direct production execution, or credential env-name ambiguity.
- Make `openudon run --dry-run` exercise the same package staging and staged digest verification as
  real handoff while still skipping final executor invocation and credential-value requirements.
- Write non-secret `openudon.run-evidence.v1` evidence for dry-runs and real handoffs with scope,
  tier, approval state, package digest, config path, staged paths, package/API paths, credential
  binding names, derived credential env names, gate outcomes, and executor invocation status.
- Keep real side effects outside OpenUdon; production still requires `approved_for_production`, a
  trusted operator environment, and a configured external executor.

Acceptance: `openudon run --dry-run` validates approval, handoff, quality, package inventory, run
config, staging, and staged digest without credentials or executor invocation; non-dry runs preserve
credential and executor enforcement; run evidence is non-secret and auditable; focused tests cover
approval/digest/config/dry-run/sandbox-production behavior. Status: implemented; see
[status-M31.md](status-M31.md).

### 32. Fnct Helper Contract Authoring

Add OpenUdon authoring support for public pure `fnct` helper selectors that
are implemented outside OpenUdon and registered by trusted runtimes. M32 starts
with Gmail raw-message rendering and keeps OpenUdon's boundary at
understanding, review, and package artifact generation.

Goal:

- Import helper descriptors from `github.com/OpenUdon/apitools/helper/...` for
  authoring and review metadata only.
- For the weather-to-Gmail pattern, preserve the local step name
  `render_weather_report` while selecting `function = "gmail.render_raw"`.
- Pass helper inputs through the operation request body and omit
  `x-uws-runtime.arguments` for request-body-object helpers.
- Require an explicit recipient input when the user says “to me” or otherwise
  does not provide an address, instead of hardcoding a personal address.
- Keep Gmail send as the side-effectful HTTP/API step; the helper is pure
  payload shaping and is not an API source operation.

Acceptance: iCoT-generated and build-authored weather/Gmail packages use
`gmail.render_raw`, map Gmail `raw` from `render_weather_report.received_body`,
pass quality gates from `intent.hcl`, preserve credential-binding checks, and
focused/full Go tests plus diff checks pass. Status: implemented; see
[status-M32.md](status-M32.md).

### 33. Runtime Data-File Handoff

Make declared workflow inputs executable through reviewed, non-secret package data instead of host
environment variables, while preserving credential values outside package artifacts.

Acceptance: `openudon build` emits `expected/data.hcl` with reviewed input placeholders and env
markers, package digests and run configs include data files, `inputs.*` lowers to executable
`variables.inputs.*`, and focused tests cover package generation and trusted-runner handoff.
Status: implemented; see [status-M33.md](status-M33.md).

### 34. Eval Seed/Build Matrix

Turn the curated eval corpus into an explicit deterministic seed/build contract. Each
`examples/eval/*/reference/policy.json` declares whether iCoT reference seeding plus build should
pass, fail for an expected negative reason, or remain advisory evidence with an allowed failure.

Acceptance: every eval fixture has a `seed_build` policy; strict positive and advisory fixtures seed
and build green from package-local artifacts; expected-negative failures are classified instead of
hidden in smoke-script output; and the matrix runs without LLM calls or provider network retrieval.
Status: implemented; see [status-M34.md](status-M34.md).

### 35. v0.1.2 Release Readiness

Prepare the public v0.1.2 release surface after the eval matrix, runtime data-file handoff, fnct
helper authoring, strict clarification, and security-overlay materialization work. Keep this
milestone focused on provider-free release gates, documentation polish, cross-repo compatibility,
and package hygiene.

Acceptance: the selected v0.1.2 release command set is documented and passes locally; docs describe
the eval seed/build matrix and runtime handoff behavior without overstating execution guarantees;
OpenUdon remains free of private `../udon` imports; and remaining v0.1.3/v0.1.4 ideas stay deferred.
Status: implemented; see [status-M35.md](status-M35.md).

### 36. Slack Smoke And v0.1.2 Tag Gate

Run one manual, redacted, real Slack sandbox smoke through the trusted-runner handoff before tagging
`v0.1.2`. Keep this milestone narrow: prove the reviewed package handoff can reach a real Slack
sandbox channel with operator-owned credentials and a reviewed executor, without adding live-provider
automation to the provider-free release gate.

Acceptance: M35 provider-free gates remain green; a Slack smoke package is reviewed, approved for
sandbox, dry-run validated, and then executed manually through `openudon run` plus a trusted
executor; evidence records only non-secret command/result metadata, package digest, Slack channel
identifier or redacted channel label, response status, timestamp, executor version/path, and any
operator notes; no Slack token, raw private message content, approval JSON, run config, or provider
output is committed; and `v0.1.2` is tagged only after the smoke passes or an explicit release
decision records why it is skipped.
Status: implemented; see [status-M36.md](status-M36.md).

### 37. Product Smoke Matrix And v0.1.2-a.1 Readiness

Add a broad product smoke matrix covering major OpenUdon artifacts, with provider-free package
coverage, required Slack live smoke, optional-provider skip evidence, and local stub-backed live
scenarios.

Acceptance: provider-free gates pass; full product smoke live passes with operator-owned Slack and
weather credentials; manual-provider cases are recorded as explicit skips; and `v0.1.2-a.1` tag
evidence is recorded. Status: implemented; see [status-M37.md](status-M37.md).

### 38. iCoT Scorecard And Agent Mode

Make iCoT reliability measurable and easier to call from local agents without adding MCP. Add
noninteractive authoring, structured lint reports, and a provider-free reliability scorecard over
the eval corpus.

Acceptance: `icot --agent --json`, `icot lint --json`, and `icot scorecard` produce versioned JSON
reports; the scorecard matches the existing seed/build policy outcomes without LLM or provider
network calls; focused iCoT tests and docs checks pass. Status: implemented; see
[status-M38.md](status-M38.md).

### 39. iCoT Request Mapping Repair And Decision Evidence

Improve iCoT at the highest-risk authoring failures: request mapping, bounded repair, and
confidence/evidence-driven blocking. Keep repair narrow and review-first.

Acceptance: request-mapping prompts expose qualified fields, LLM-proposed qualified aliases are
validated before entering intent state, `icot repair` can dry-run or apply only mapping/output/
dependency repairs, source/operation/credential/side-effect mutations remain rejected, and focused
iCoT tests plus normal gates pass. Status: implemented; see [status-M39.md](status-M39.md).

### 40. iCoT Natural-Language Authoring Reliability Expansion

Expand iCoT reliability evidence from seed/build fixtures to curated natural-language variants for
Slack, Gmail, weather, and weather-to-Gmail workflows. Keep the milestone provider-free: positive
variants build from reviewed package-local artifacts, missing-detail and unsafe-negative variants
stop with declared failure families, and scorecard summaries group results by provider family,
variant class, and failure family.

Acceptance: `icot scorecard --include-variants` and `make icot-authoring-scorecard` pass without
LLM or provider calls; release SaaS checks include the authoring variant scorecard; docs explain the
LLM-authoring/deterministic-execution enterprise boundary. Status: implemented; see
[status-M40.md](status-M40.md).

### 41. UWS 1.3 Contract Baseline For AsyncAPI Source Descriptions

Establish the public UWS 1.3 contract for first-class AsyncAPI source descriptions while preserving
OpenAPI, Google Discovery, and AWS Smithy source behavior.

Acceptance: UWS 1.3 schema/model/docs define AsyncAPI source descriptions, compatibility
expectations are recorded, and OpenUdon tracks downstream package/generation work separately.
Status: implemented; see [status-M41.md](status-M41.md).

### 42. AsyncAPI Metadata Boundary

Add and consume public AsyncAPI metadata parsing through apitools without moving protocol execution
into OpenUdon.

Acceptance: apitools exposes prompt-safe AsyncAPI operation metadata, OpenUdon treats AsyncAPI as
reviewed local source metadata, and runtime protocol execution remains trusted-executor-owned.
Status: implemented; see [status-M42.md](status-M42.md).

### 43. OpenUdon Package And Generation Support For AsyncAPI

Teach OpenUdon package discovery, synthesis, quality, plan, review, and handoff logic to preserve
AsyncAPI source documents as first-class package artifacts.

Acceptance: AsyncAPI source files are staged, digested, validated, rendered in UWS source
descriptions, and carried through review/handoff evidence without OpenAPI lowering. Status:
implemented; see [status-M43.md](status-M43.md).

### 44. iCoT, Examples, And Eval Coverage For AsyncAPI

Extend authoring and eval coverage so iCoT and deterministic fixtures can select and review
AsyncAPI-backed operations.

Acceptance: examples/evals cover AsyncAPI source selection and package generation, iCoT can surface
AsyncAPI metadata as local source context, and unsupported protocol execution remains explicitly
reviewed. Status: implemented; see [status-M44.md](status-M44.md).

### 45. Release Readiness And Documentation For UWS 1.3

Close the UWS 1.3 release-readiness pass with docs, quality gates, and cross-repo compatibility
evidence for AsyncAPI source descriptions.

Acceptance: docs explain UWS 1.3 source behavior, deterministic gates pass, and release notes/status
files record remaining executor-owned protocol limitations. Status: implemented; see
[status-M45.md](status-M45.md).

### 46. UWS 1.4 Source-Family Eval Coverage

Expand OpenUdon source-family coverage for UWS 1.4 GraphQL, OpenRPC, gRPC/protobuf, and OData
metadata.

Acceptance: reviewed local source artifacts for UWS 1.4 families are covered by package, quality,
handoff evidence, and eval fixtures, while protocol execution remains trusted-executor-owned.
Status: implemented; see [status-M46.md](status-M46.md).

### 52. Ramen/OpenUdon Approval And Governance Contract Review

Compare Ramen's plan/run governance and approval artifacts with OpenUdon's
approval, review-handoff, run-config, and run-evidence artifacts before moving
any code or creating a shared module.

Goal:

- Inventory Ramen `governance`, `ramen.approval.v1`, and `ramen.run.v1`
  approval inputs.
- Inventory OpenUdon `openudon.approval.v1`,
  `apitools.review-handoff.v1`, `openudon.executor-run.v1`, and
  `openudon.run-evidence.v1`.
- Classify common approval/governance concepts separately from
  Ramen-specific desired-state bindings and OpenUdon-specific package/handoff
  bindings.
- Recommend no extraction, a narrow future governance module, or later
  executor/run/CLI review only after the real contract comparison is complete.

Acceptance: M52 produces a written comparison and recommendation without
changing Go APIs, command behavior, wire versions, imports, or module
dependencies. OpenUdon keeps approval templates, package digests,
review-handoff validation, and trusted-runner enforcement; Ramen keeps
desired-state reconciliation and Ramen-specific run/audit history.

### 57. Trusted Executor Output Contract

Define and consume a real trusted-executor output artifact before adding status
or confirmation-read records to OpenUdon async sidecars.

Goal:

- Add a non-secret udon execution report contract.
- Have OpenUdon pass a report path to compatible udon executor argv.
- Translate real reports into Evidence async status and confirmation-read
  observations.
- Preserve external-runner preflight boundaries when no report contract is
  available.

Acceptance: `openudon run` sidecars can include request, response, status, and
confirmation-read records sourced from a real udon report, while dry-run and
outer runner shim evidence remain bounded to what OpenUdon can directly attest.
Status: implemented; see [status-M57.md](status-M57.md).

### 58. Run Evidence Verifier Hardening

Harden `openudon run-evidence verify` for expanded sidecars and archival use.

Goal:

- Validate the expanded sidecar record set and schema.
- Fail closed on missing files, duplicate refs, invalid purposes, bad versions,
  and malformed observation payloads.
- Prove copied archive bundles verify from their archive directory.

Acceptance: archived run-evidence bundles and async sidecars verify when intact
and fail with actionable errors when sidecar references or observation records
are malformed. Status: implemented; see [status-M58.md](status-M58.md).

### 59. Async Evidence Release Candidate Pass

Produce deterministic release evidence for the executor report and expanded
async sidecar story.

Goal:

- Update release guidance and templates for executor reports and sidecar
  verification.
- Run provider-free release gates and record results.
- Keep release artifacts local/ignored unless explicitly promoted.

Acceptance: the release candidate evidence path demonstrates package handoff,
executor report ingestion, async sidecar archive verification, and provider-free
release gates without tagging or pushing by default. Status: implemented; see
[status-M59.md](status-M59.md).

### 60. Executor-Report Archive Command

Add `openudon run-evidence archive` for local release evidence collection.

Goal:

- Copy `run-evidence.json`, referenced async sidecars, and an executor report
  when the trusted executor produced one.
- Keep copied async sidecar paths compatible with archive-local verification.
- Verify the archived bundle after copying.

Acceptance: archive output verifies with `openudon run-evidence verify` from
the archive directory and fails closed on invalid source evidence. Status:
implemented; see [status-M60.md](status-M60.md).

### 61. Release-Note Generation

Add a local release-note draft command for run evidence.

Goal:

- Write a Markdown draft with current commit, gate results, verifier output,
  package digest, and evidence/report paths.
- Keep the draft local and explicit; it does not tag, publish, or infer release
  approval.

Acceptance: `openudon release-notes draft` writes a reproducible evidence note
from verified run evidence and supplied gate results. Status: implemented; see
[status-M61.md](status-M61.md).

### 62. Real Executor Smoke With Local Udon

Add a provider-free non-dry-run smoke path against a sibling-built udon binary.

Goal:

- Build `../udon/cmd/udon` into a local ignored workdir.
- Materialize the runtime-only eval seed into a package and run it through
  OpenUdon's trusted handoff with `OPENUDON_EXECUTOR` pointing at the local
  binary.
- Verify run evidence includes the udon executor report and expanded async
  status/read observations.

Acceptance: `openudon local-udon-smoke` produces a provider-free non-dry-run
summary, `executor-report.json`, `run-evidence.json`, and expanded
`async-evidence.json` without importing udon as a Go dependency. Status:
implemented; see [status-M62.md](status-M62.md).

### 63. Verifier Schema Polish

Add JSON Schema coverage for emitted async sidecars.

Goal:

- Validate generated `openudon.async-evidence-bundle.v1` sidecars against the
  checked-in schema in tests.
- Keep Go-struct validation as the runtime verifier and schema validation as an
  emitted-contract regression test.

Acceptance: focused tests prove emitted async sidecars satisfy the documented
schema in addition to strict Go validation. Status: implemented; see
[status-M63.md](status-M63.md).

### 64. Release Evidence Workflow Consolidation

Consolidate the release-evidence helper commands into a single repeatable
operator flow.

Goal:

- Add one command or make target that runs the provider-free release evidence
  sequence: local udon smoke, run-evidence archive verification, release-note
  draft generation, and deterministic gate result capture.
- Emit one compact machine-readable summary that records command paths,
  archive paths, release-note path, commit, gate status, and verifier output.
- Keep all evidence local/ignored by default and avoid tagging or publishing.

Acceptance: an operator can produce the M60-M63 release evidence bundle with one
documented command, inspect a summary JSON/Markdown pair, and rerun verification
against the archive. Status: implemented; see [status-M64.md](status-M64.md).

### 69. CLI-First Public Beta

Publish OpenUdon's first public release after repairing the standalone
Authoring lifecycle dependency.

Goal:

- Define a narrow v0.1.x compatibility promise for deterministic package,
  approval, trusted-handoff, and run-evidence commands and artifacts.
- Keep iCoT/LLM/provider behavior and maintainer helpers experimental.
- Prove public-module operation without a parent workspace or sibling
  checkout.
- Publish co-versioned `openudon`, `icot`, and `udon-runner` archives for six
  platform targets with checksums and local build metadata.

Acceptance: clean source-archive and standalone gates pass; the comprehensive
provider-free local gates and sibling udon smoke pass; public main CI is green;
annotated tag `v0.1.0` publishes six verified archives plus `SHA256SUMS`; an
isolated tagged install reports version 0.1.0 and completes the credential-free
author/build/assess/approval/dry-run path. Status: implemented; see
[status-M69.md](status-M69.md).

### 70. Adaptive Evidence-Grounded iCoT v2

Replace the fixed, one-blocker-at-a-time iCoT loop with dependency-aware
frontier rounds grounded in inspected source evidence.

Goal:

- Select one confirmed active workflow from broad requests and preserve later
  candidates without implementation breakdown.
- Use Authoring's versioned interview graph and unlimited round engine, with
  structured technical deferrals and three-round no-progress diagnosis.
- Inspect bounded local API sources before questioning, keep remote lookup
  explicitly approved and bounded, and stage reviewed sources only in the
  approved artifact transaction.
- Migrate OpenUdon sessions, transcripts, reports, scorecards, replay/eval, and
  variants to v2 without a v1 decoder.
- Support complete approval, approved incomplete drafts, atomic promotion,
  collision backup/rollback, candidate-workflow project sections, and a
  read-only agent surface.

Acceptance: more than 20 decisions can complete in one or more deterministic
frontier rounds; all eight source families and local safety/limit cases are
covered; remote deny/approval/timeout/unsafe/empty/success cases are offline;
cancel/print write nothing; draft/final/source collision and promotion behavior
is atomic; full/normal/fast and agent behavior match the v2 contract; v1 inputs
are rejected; migrated support commands and offline release gates pass. Status:
implemented; see [status-M70.md](status-M70.md).

## Closed XRD Regression Matrix

| ID | Status | Regression owner | Boundary |
| --- | --- | --- | --- |
| XRD-001 | Closed | udon / OpenUdon | Structured output fallback regressions only. |
| XRD-002 | Closed | udon / OpenUdon | UWS structural/action preservation regressions only. |
| XRD-003 | Closed | UWS / udon / OpenUdon | UWS 1.1 timeout/idempotency public contract is done; runtime enforcement remains udon work. |
| XRD-004 | Closed | OpenUdon eval first, udon if needed | Pagination, request bodies, security, writes, response extraction, and multi-service eval coverage. |
| XRD-005 | Closed | OpenUdon / optional orchestration owner | OpenUdon emits handoff evidence and trusted wrapper; external orchestration owns managed routing if needed. |
| XRD-006 | Closed | OpenUdon release owner / provider owners | Provider drift reporting in eval evidence and release notes. |
| XRD-007 | Closed | OpenUdon / infra | Local readiness reporting; hosted automation deferred. |
| XRD-008 | Closed | OpenUdon eval first, UWS/udon if needed | Runtime/profile coverage as policy/eval evidence. |
| XRD-009 | Closed | OpenUdon release owner | Expanded-corpus minimum brief release gate. |

## Runtime/Profile Eval Coverage Detail

Runtime/profile semantics and generic execution remain upstream in `../udon` or `../uws`. OpenUdon's
XRD-008 coverage is policy language, reference intent shape, review evidence, and fixture
regression only.

| Category | Fixtures | Boundary proven |
| --- | --- | --- |
| Approved function runtime | `runtime-only-render`, `support-priority-routing`, `profile-boundary-manifest` | Trusted local adapters and renderers can be generated when declared in project policy and Function Contracts. |
| Approved command runtime | `cmd-allowed-deploy` | `cmd` is allowed only when project policy explicitly permits a sandbox command. |
| Denied command runtime | `cmd-disallowed-deploy` | A generated `cmd` step remains a policy failure when the project denies command execution. |
| Denied SSH/runtime profiles | `cmd-disallowed-deploy`, `profile-boundary-manifest` | OpenUdon policy keeps `ssh` and future profile runtimes out of generated intent unless an upstream public/runtime contract exists and project policy approves it. |
| Future profile boundary | `profile-boundary-manifest` | SQL-style profile work is represented as a trusted `fnct` manifest request instead of inventing `sql`, `ssh`, or `x-udon-*` runtime semantics in OpenUdon. |

OpenUdon reference intents must not invent unsupported runtime types such as `sql`, `smtp`, `llm`, or
profile-specific `x-udon-*` payloads. Any future fixture that needs real profile semantics should
open upstream work in `../udon` or `../uws` before OpenUdon prompt defaults emit those fields.
HTTP is not a runtime-profile type in UWS extension payloads: HTTP/OpenAPI operations must bind
through core UWS OpenAPI fields, and `type: http` in public `x-uws-runtime` payloads is rejected.
The public runtime supplement is intentionally small: runtime selector fields only, with no
provider/security configuration, HTTP metadata, or request/response schema projection. Udon's legacy
private `x-udon-runtime` remains a compatibility surface until a separate udon DTO and export
migration removes its raw HTTP projection.

### 67. Desired-State Conversion Removal

Remove retired desired-state conversion material from OpenUdon now that Ramen
owns conversion, planning, reconciliation, and provider/resource operation
mapping.

Acceptance: OpenUdon has no conversion docs, status-file tracks, fixtures,
command docs, parser/conversion dependencies, or provider conversion roadmap.
OpenUdon keeps only negative boundary checks that reject parser/conversion
imports. Status: implemented; see [status-M67.md](status-M67.md).

## Milestone Scope Supplements

These rows restore concise scope and acceptance ownership for completed status
files whose detailed implementation ledgers were added after the original
numbered roadmap prose.

| ID | Goal | Acceptance |
|---|---|---|
| B01 | Establish the initial product, module, memory-bank, evolution, workspace, and sibling boundaries. | The public module and private harness exist with documented ownership and no runtime side effects. |
| M27 | Adopt UWS 1.2 first-class OpenAPI, Google Discovery, and AWS Smithy source descriptions. | Typed source paths package, validate, and hand off without moving protocol/auth execution into OpenUdon. |
| M53 | Consume shared Authoring interactive iCoT mechanics. | CLI behavior stays stable while product prompts, intent, reports, artifacts, and clients remain downstream. |
| M54 | Forward neutral async execution evidence from trusted-runner handoff. | Sidecars are digest-bound and secret-free and do not interpret Ramen convergence or state. |
| M55 | Harden operator and archive usability of async evidence. | Relative refs, CLI paths, failure coverage, docs, and relocation checks pass. |
| M56 | Add strict run-evidence verification and release readiness. | CLI/schema validation fails closed while preserving the executor boundary. |
| M65 | Close bounded iCoT M29 response-metadata, read-only prework, and replay-gate follow-ups. | Added operations and mappings remain metadata-backed, unambiguous, input-free, and non-side-effectful. |
| M66 | Review provider expansion ownership after Ramen migration. | Conversion stays in Ramen; only OpenUdon-owned review/package evidence can return here. |
| M68 | Align lifecycle hints and operation-detail planning. | One deterministic per-round plan yields source-correct, bounded hints and detail refs. |

### M71 Parallel-Lane Harness Migration

**Goal.** Adopt permanent domain lanes and runner-compatible task ledgers
without changing OpenUdon product behavior.

Acceptance:

- Every status file is canonical, permanently indexed, and parseable.
- Legacy mappings and restored M07/M08 history are explicit.
- Future authoring, package, and eval work has non-overlapping lane ownership.
- Candidates remain unnumbered until promotion.
- Structural, full/standalone, boundary, document-memory, release, and
  unattended no-action checks pass.

### M72 Structured API Security Alternatives

**Goal.** Preserve Apitools security alternatives through OpenUdon authoring
without unioning credentials or losing resume-time safety evidence.

Acceptance:

- Every operation retains outer OR alternatives, inner AND requirements, and
  explicit anonymous access in prompts and shared context.
- iCoT forces one stable selection before mappings and rejects fields from
  unselected alternatives; the choice and evidence survive resume.
- Incomplete local discovery blocks immediately, prompt-budget compaction is
  a visible deferable technical blocker, network approval survives discovery,
  and source target collisions fail before the write transaction.

### M73 Shared Interview Settlement And Lifecycle-Ranking Boundary

**Goal.** Remove OpenUdon's duplicate generic frontier settlement and consume
generic lifecycle-role ranking from its owning metadata module.

Acceptance:

- Terminal, runtime, and UI-capable paths share Authoring's exact-frontier,
  clone-before-mutation settlement invariants while OpenUdon retains graph,
  prompt, readiness, intent, and artifact policy.
- Lifecycle ranking consumes source-bearing Apitools operation summaries,
  preserves duplicate operation IDs across documents, and scopes Google
  Discovery upload normalization to explicit provenance.
- Workspace tests, vet, the iCoT scorecard, compatibility checks, document
  checks, and published-pin standalone test/vet pass.

### A01 API-First Browser-Profile Fallback Authoring

**Goal.** Let iCoT build reviewable UWS workflows for UI-only capabilities by
selecting verified Browsertools profiles only when inspected API sources do not
cover the active workflow.

Acceptance:

- `icot` accepts repeatable `--browser-profile ID=PATH` and
  `--browser-registry URL`; existing `--source-root` discovers both API-family
  sources through Apitools and browser profiles/bundles through Browsertools.
- Local examples and explicit inputs are inspected before questions. API
  candidates remain preferred; a browser source becomes eligible only after
  the active capability lacks a matching API operation or the user explicitly
  chooses the reviewed browser route.
- Remote browser-registry lookup uses the existing network policy, requires
  approval in interactive `ask` mode, is disabled in agent mode unless
  explicitly allowed, and preserves all Browsertools bounds and blockers.
- The interview orders browser source before action and action before mappings,
  session requirement, side effects, outputs, and final approval. Browser
  side-effect posture and approval may not be deferred.
- Agent mode returns the complete frontier, API and browser candidates,
  lifecycle/expiry blockers, origin/action evidence, digests, and proposed file
  actions without prompting or writing.
- Approved transactions materialize verified profiles under
  `browser-profiles/`, safe review metadata under `.icot/`, project candidate
  text, and browser-bound UWS intent atomically. Print/cancel/failed approval
  write no deliverables; collisions, backups, drafts, resume/promotion, and
  rollback follow the existing v2 lifecycle.
- Project quality, lint, package inventory, digest, trusted staging, and run
  handoff cover browser profile inputs while rejecting raw captures, expired or
  revoked bundles, invented actions, unconfirmed mutations, and unsafe
  placeholder promotion.
- Public docs and evals cover all prompt modes, local/remote/no-network paths,
  API preference, read-only browser workflows, confirmed mutations, and Udon
  handoff using only local test servers.

### A02 Additive Browser Authentication And Named Sessions

**Goal.** Let iCoT include a reviewed website sign-in and MFA flow before a
protected `uws.browser.1.5` action without changing legacy browser workflows or
moving private runtime state into OpenUdon.

Acceptance:

- Browsertools discovery identifies local, secret-free
  `uws.browser-authentication.1.0` profiles as a distinct source family;
  OpenUdon stages them only after proposal approval and does not publish them to
  the static capability registry.
- The frontier selects an exact authentication flow, execution-local named
  session, exact symbolic credential-slot mappings, timeout no greater than 600
  seconds, and non-deferrable step-specific authoring approval.
- A protected browser action uses the same named session after the
  authentication step. Existing legacy opaque session posture and all
  `uws.browser.1.5` capability documents remain compatible.
- Intent lowers authentication to
  `uws.browser-authentication-call.1.0`, lowers the protected action's named
  session supplement, and emits UWS 1.7 without importing Udon.
- `.icot/browser-authentication.json`, package inventory, expected plan,
  quality, review, and handoff evidence preserve profile digests, flows,
  origins, expiry, symbolic mappings, session names, and approvals while
  rejecting stale/tampered profiles, invented flows, mismatched mappings,
  dangling sessions, missing timeout/approval, and secret values.
- Enrollment, recovery, password changes, consent, logout, account creation,
  CAPTCHA solving, credentials, MFA responses, cookies/storage state, drivers,
  and live sessions remain outside OpenUdon.

### A03 Browsertools Authoring Handoff And Guided-Result Consumption

**Goal.** Let iCoT coordinate the missing-profile authoring boundary and
consume a reviewed Browsertools guided result without becoming a browser,
credential, session, or raw-evidence owner.

Dependencies: Browsertools E03/P03/A02/E04, OpenUdon A01/A02, and the unchanged
UWS browser.1.5/browser-authentication contracts.

Acceptance:

- `icot browser-authoring plan` emits exact
  `openudon.browser-authoring-handoff.v1` JSON from an explicit clean target
  URL, exact origins, profile/action IDs, login posture, and a private root
  disjoint from the example. The root is an existing restrictive non-symlink
  directory, and file output stays inside it with new-only mode `0600`.
- The plan contains typed argv templates and manual gates but performs no external
  command, network request, browser installation/launch, credential-environment
  read, profile inference, source copy, or deliverable write. Agent JSON can
  carry the same handoff without changing its read-only behavior.
- An unauthenticated plan covers doctor, finite private capture, exact-ID
  export, local raw review/redaction, normalized evidence, deterministic guided
  authoring, and the explicit iCoT resume input. Raw/private/rich evidence and
  credential/session material are rejected as OpenUdon inputs.
- A login-required plan returns `needs_reviewed_profiles` and no executable
  capture argv because Browsertools cannot transfer an authenticated authoring
  context into capability capture. It requires separately reviewed
  authentication and capability profiles and names current popup/iframe
  SSO/CAPTCHA/enrollment/recovery/consent exclusions.
- Explicit `browsertools.guided-authoring.v1` files are strict-decoded, replayed
  through Browsertools draft construction, checked for exact decision/profile
  equality, reverified against their review at current time, and reduced to
  canonical `uws.browser.1.5` bytes. Unknown fields, tampering, stale review,
  broad-root implicit promotion, and envelope/evidence staging fail closed.
- Focused/full tests, vet, standalone module gates, boundary/doc-memory checks,
  and cross-repo Browsertools/UWS/Udon compatibility gates pass before commit.

### A04 Explicit Authenticated Goal-Directed Browser Authoring

**Goal.** Let an operator explicitly keep one Browsertools-owned headed
Chromium context alive across human login/MFA and post-login exploration while
iCoT guides only reduced, reviewed semantics toward a typed and human-confirmed
goal.

Dependencies: Browsertools A03/E05/P04, UWS 1.8 browser contexts,
Browserdriver M03, Udon M29, and OpenUdon A03/P01/E01.

Acceptance:

- `icot browser-author live` validates an absolute non-symlink Browsertools
  binary, restrictive disjoint private root, exact clean URLs/origins, typed
  after-authentication continuation, and typed accessibility completion
  predicate before launching `browsertools author-session chromium`.
- Normal iCoT, agent mode, and `browser-authoring plan` remain non-executing.
  The live child receives an allowlisted environment without credential or LLM
  provider variables; Browsertools alone owns Playwright-Go and the
  non-persistent context.
- The strict bounded `browsertools.author-session.v1` client accepts only
  reduced origin/path/context/candidate semantics. Before any model receives
  them, iCoT identifies the provider/model and requires once-per-run disclosure
  approval; denial falls back to human guidance. Candidate actions never carry
  selectors, coordinates, JavaScript, values, DOM, screenshots, or browser
  handles.
- Typed-goal review, API-first browser override, new-origin, click/POST,
  credential/MFA attention, typed-plus-human completion, and final staging are
  distinct human gates; `--yes` bypasses none. Unknown messages, denial,
  timeout, crash, origin escape, ambiguous targets, malformed output, or
  incomplete teardown imports nothing.
- OpenUdon stable-reads and digest-checks the private deterministic
  `browsertools.authenticated-authoring.v1` envelope, independently validates
  its bounds, trace, origins/contexts, goal proof, reviews, freshness, secret
  absence, and embedded UWS profiles, and keeps the envelope outside the
  package. Final approval atomically stages only canonical profiles and safe
  `.icot` metadata; normal iCoT performs later flow/action/session authoring.
- Existing browser 1.5/authentication 1.0 sources still emit UWS 1.7 and call
  1.0. Browser 1.6 or authentication 1.1 sources select UWS 1.8 and call 1.1 as
  required. Playwright-Go authoring remains separate from Udon/Browserdriver
  trusted runtime replay.

Status: implemented, reviewed, and verified in OpenUdon commit `7b516e0`; see
[status-A04.md](status-A04.md).

### A05 Live Browser Observation Hardening

**Goal.** Independently verify that Browsertools applied its canonical label
policy, reject substituted/noncanonical observations without disclosing their
content, and enforce the exact candidate authority negotiated at session start.

Dependencies: OpenUdon A04/E03 and published Browsertools E06 commit
`55f40e56e03e2ce878521a50bc38a3674e18defc` at
`v0.0.0-20260817000231-55f40e56e03e`.

Acceptance:

- OpenUdon accepts a candidate label only when applying
  `authorsession.ReduceAccessibilityLabel` returns that exact incoming value;
  no duplicated phrase list or direct generic redaction check remains.
- Candidate validation is ordered by ID syntax, duplication, role, match
  count, and canonical label. Local rejection diagnostics contain a valid safe
  candidate ID plus a closed reason, while malformed IDs and rejected page
  text never reach stderr, human display, or the planner.
- The protocol reader starts with the absolute 512 ceiling and switches to the
  requested `MaxCandidates: 128` immediately after `start`; exactly 128
  candidates are accepted, while 129 candidates or `matches > 128` fail before
  display or disclosure.
- Suspicious genuine Browsertools observations continue with fixed markers.
  Planner output remains a closed typed action over observed unique candidate
  IDs, canonical same-origin GET navigation, or human fallback; selectors,
  scripts, coordinates, URL authority, and input values cannot be invented.
- The exact published Browsertools pin resolves normally. Workspace and
  standalone tests/vet, OpenUdon checks, doc-memory, strict docs, integration
  matrix, link/diff checks, and clean-revision qualification pass.

Status: complete; see [status-A05.md](status-A05.md).

### A06 Typed MFA And Dashboard Output Authoring

**Goal.** Replace live MFA/output inference with strict human-reviewed v2
choices and independently validate the resulting portable profiles.

Dependencies: UWS 1.9/browser 1.7, Browserdriver M05, Udon M32, and
Browsertools A04.

Acceptance:

- OpenUdon rejects author-session/result v1, requires the v2 MFA/output
  capabilities, and requests exactly `MaxOutputs: 16`.
- The planner remains limited to focus, observed click, bounded same-origin
  GET navigation, or human fallback. One exact compatible MFA kind and every
  final output declaration remain human-only.
- Completion accepts zero through 16 current-observation string, integer,
  number, Boolean, or presence outputs with safe unique keys and exact-name or
  unique-role locator modes, then prints a value-free summary and requires
  confirmation.
- Returned challenge kinds, symbolic credential slots, selection identities,
  context and match proofs, review digests, profile versions, and profile
  contents are validated independently. Substitution, malformed IDs, rejected
  labels, stale/ambiguous targets, secret-shaped keys, and v1 input fail closed.
- Browser 1.7 selects UWS 1.9. Browser 1.5/1.6 and authentication 1.1 retain
  their existing UWS 1.7/1.8 selection behavior.
- Standalone/release/docs gates and the clean five-repository provider-free
  matrix pass before publication.

Status: complete in OpenUdon commit
`6a08b81317a852b9b8581c502eadbbdf591508e1`; see
[status-A06.md](status-A06.md).

### A07 Headless iCoT Authoring Engine

**Goal.** Add the internal authoring seam needed by a future local UI without
changing terminal behavior or publishing a new service/API contract.

Dependencies: M53, M70, A01-A06, and P01.

Acceptance:

- `internal/icot/engine` opens empty, seeded, explicit-session, and resumable
  draft state; discovers reviewed API/browser sources; and returns a
  JSON-marshalable snapshot containing the complete frontier, readiness and top
  issue, active boundary, candidates, selected/source candidates, exact file
  actions, evidence, and an in-memory render preview when available.
- One engine round must answer the entire current dependency-ready frontier
  and is applied through `elicitor.ApplyFrontierRound`; duplicate,
  non-frontier, blank, and malformed deferral answers fail without partial
  mutation. Successful rounds normalize, resynchronize source plans and
  browser verification, and autosave `.icot/session.yaml`.
- Preview never writes final deliverables. Approval requires explicit human
  input and shares terminal iCoT's source revalidation/materialization, browser
  metadata, collision handling, rollback-capable atomic writer, and draft
  cleanup behavior.
- The evidence ledger round-trips annotations, assumptions, mapping
  classifications, decision evidence, draft operation refs, and structured
  draft events across autosave/resume.
- Tests prove byte-for-byte `project.md` and `workflows/intent.hcl` parity with
  terminal authoring for `examples/eval/runtime-only-render` plus browser
  verification revalidation and safe metadata retention.
- Phase A adds no `icot ui`, web server, React frontend, folder browsing,
  published JSON schemas, or `icot session` CLI verbs.

Status: complete; see [status-A07.md](status-A07.md).

### A08 Local iCoT UI Server

**Goal.** Add the Phase B single-workspace local server over A07 so a future
authoring frontend can inspect and drive the engine without widening execution
or remote-service authority.

Dependencies: A07, M70, A01-A06, and P01.

Acceptance:

- `icot ui --example DIR` accepts explicit session/example seeds, reviewed
  API/browser sources, network policy, loopback port, and no-open policy.
  Startup precedence is explicit seed, resumable session, existing final
  project/intent, then empty state.
- One stdlib-only server owns one engine, cached snapshot, SHA-256 revision,
  completion bit, and optional write result. Every mutation is serialized
  under one lock and requires the exact current revision; two concurrent
  same-revision mutations have exactly one winner.
- The experimental `openudon.icot-ui-api.v1` API exposes health, snapshot,
  complete-frontier round, and explicit approval routes. Round clients cannot
  provide slots or evidence sources. Strict one-document JSON rejects unknown
  fields and bodies over 1 MiB with structured errors.
- A fresh 256-bit capability token bootstraps an HttpOnly SameSite=Strict
  cookie beneath an unguessable per-process path so sibling loopback services
  do not receive or overwrite it; bearer auth is also accepted. The server
  binds only `127.0.0.1`, requires its exact Host and any supplied Origin,
  emits no CORS permission, and sets restrictive no-store, CSP, framing,
  referrer, content-type, and permissions headers.
- Approved final and incomplete writes freeze mutation while preserving
  inspection. Source revalidation, collision/draft policy, and atomic writes
  remain in A07. Replaced verification and unavailable registry failures cross
  HTTP without writes or revision corruption. Mutation errors cannot leave an
  advertised stale revision, and operational context/filesystem failures are
  distinguished from domain rejections.
- Separate embedded HTML, JavaScript, and CSS assets form a read-only shell
  showing workspace paths, readiness/top issue, frontier count, preview,
  completion, and formatted snapshot JSON. Phase B adds no React/Node, folder
  browser, remote serving, TLS/account system, multi-session hosting, LLM
  drafting, execution, or live browser-authoring authority.

Status: complete; see [status-A08.md](status-A08.md).

### A09 Enhanced Phase B Reliability And Status UX

**Goal.** Close the remaining mutation-lifecycle, optimistic workspace,
transport-hardening, and read-only status UX gaps without widening Phase B
beyond one loopback workspace and one trusted operator.

Dependencies: A07, A08, M70, A01-A06, and P01.

Acceptance:

- `ApplyRound` constructs prospective state and snapshot before atomic draft
  persistence; errors preserve memory and draft bytes, while cancellation after
  persistence starts cannot interrupt finalization.
- Approval returns one exact `ApprovalResult` containing the approved snapshot
  and write result, performs no fallible post-commit refresh, and treats empty
  directory pruning as best effort.
- Engine failures are typed as rejected, conflict, operational, or
  indeterminate. The shared writer cleans temporary backups after success or a
  completed rollback, reports post-commit cleanup failure as a non-fatal write
  warning, reports rollback failure as indeterminate, and rejects outputs
  outside the canonical example root or through descendant symlinks.
- Sorted SHA-256 fingerprints cover fixed project/draft/metadata paths,
  selected materialized sources, and proposed actions. Paths newly selected by
  a mutation are bound to a pre-refresh workspace observation. External edits
  or a competing engine latch `externally_modified`, change the revision,
  preserve cached inspection, and block mutation until restart;
  `allow_overwrite` does not bypass drift.
- Only `/api/v2/snapshot`, `/api/v2/round`, and `/api/v2/approve` are served.
  Conditional GET, exact bootstrap scope, strict media/charset and recursive
  duplicate-key validation, invalid-UTF-8 rejection, 1 MiB body and 32 KiB
  header limits, read timeouts, typed statuses, retryable request-ID errors,
  COOP/CORP, and sanitized 500-only logging are covered through a real
  loopback listener.
- The separate embedded shell polls every two seconds only while visible,
  refreshes immediately on visibility, backs off to 30 seconds, retains cached
  JSON after errors, and displays revision, refresh time, source/action counts,
  readiness, issue, frontier, preview, completion, and restart-required drift
  state without mutation controls or new frontend dependencies.

Status: complete; see [status-A09.md](status-A09.md).

### A10 Interactive Phase C Authoring And Review Shell

**Goal.** Turn the A09 status shell into the first interactive browser
authoring/review experience without widening the local trusted-operator or
execution boundary.

Dependencies: A07-A09, M70, A01-A06, and P01.

Acceptance:

- The full current frontier is rendered as accessible required form controls
  with stable labels, rationale, slot context, and optional explicit
  recommendation-fill actions.
- A round submits every current question and the exact displayed revision;
  incomplete forms focus the first missing answer and mutations are never
  automatically retried.
- The shell displays selected sources, readiness, top issue, exact project and
  intent previews, proposed actions, and sorted read-only write conflicts
  before approval.
- Review acknowledgement and overwrite acknowledgement are distinct, with
  separate explicit final and incomplete approval controls that preserve the
  engine's exact flags and frozen result.
- Domain rejection remains editable; retryable failure reconciles before an
  explicit retry; stale state preserves unsent answers until explicit adoption;
  workspace drift requires restart; indeterminate failure locks mutation; and
  frozen completion remains inspectable.
- Complete prepared plans reject reserved source targets plus duplicate,
  case-insensitive-equivalent, ancestor/descendant, and remove/write output
  collisions before filesystem mutation. Pre-refresh observation is bounded to
  watched/current-candidate paths and hashes files by a cancellable stable-file
  stream.
- Successful mutations announce status politely and focus the first question
  in the next frontier, the proposal-review heading when approval becomes
  available, or the completion banner after approval.
- Production remains embedded HTML/CSS/JavaScript with no React/Node, folder
  browser, accounts, remote/LAN serving, workflow execution, UI-owned LLM
  drafting, or live browser orchestration.

Status: complete; see [status-A10.md](status-A10.md).

### A11 Phase C Real-Browser Qualification

**Goal.** Make the interactive Phase C lifecycle a required provider-free
release property proven in a real browser against the production listener.

Dependencies: A10 and the existing sandboxed Chromium release-runner setup.

Acceptance:

- A build-tagged Playwright-Go suite launches Chromium against a real
  loopback `icot ui` handler; production binaries retain no browser-runtime
  dependency.
- Browser journeys cover accessible names, keyboard order and visible focus,
  recommendation fill, complete rounds, preview/action/conflict display,
  overwrite review, and both final and incomplete approval modes.
- Lifecycle cases cover stale unsent-input preservation and explicit adoption,
  workspace drift, retryable and rejected failures, indeterminate lockout,
  completion freeze, conditional `304`, two-second visible polling,
  visibility refresh, and exponential backoff capped at 30 seconds.
- Layout remains usable at 360 CSS pixels and 200-percent zoom without page
  horizontal overflow.
- `make icot-ui-browser-check` is required by `make release-saas-check` and tag
  automation, with Chromium sandboxing retained on the release runner.

Status: complete; the E09 release qualification passed all 13 journeys with
the Chromium sandbox required and enabled. The diagnostic sandbox-disable
override did not contribute to that evidence. See [status-A11.md](status-A11.md).

### P01 Value-Free Browser Verification Package And Review Evidence

**Goal.** Bind optional Browsertools current-page and cross-engine verification
facts to the reviewed OpenUdon package without treating private browser data as
package evidence or making portability a universal execution requirement.

Dependencies: A03 and Browsertools P03/E04.

Acceptance:

- Explicit value-free `browsertools.live-check.v1` and
  `browsertools.portability-check.v1` files can be attached to an exact staged
  browser profile/action set; profile digest, origin, action, engine, timestamp,
  and fixed-diagnostic consistency are revalidated before package approval.
- Input decoding and cross-report matching are bounded and deduplicated, and
  safe summaries are independently derived rather than trusted merely because
  an unsigned report is internally consistent.
- Safe summaries and source digests enter `.icot` review metadata, quality,
  human review, package inventory, canonical digest, and trusted handoff.
  The Browsertools reports may be staged only if their schemas remain
  value-free; page values and backend errors remain impossible by contract.
- Private raw cache entries, guided evidence envelopes, rich screenshot/trace/
  HAR bundles, assisted-login session material, unknown report fields, stale or
  mismatched facts, and invented engine success fail closed.
- Portability evidence is opt-in review confidence, not a requirement for all
  browser workflows and not permission to rewrite locators or change
  browser.1.5.

Status: implemented in OpenUdon commit `81ede87`; see
[status-P01.md](status-P01.md).

### E01 Browser Authoring-To-Handoff Integration Evaluation

**Goal.** Make the full supported API-first/browser-authoring/package path and
its unsupported authenticated-capture boundary measurable with provider-free,
non-executing release evidence.

Dependencies: A03 and P01.

Acceptance:

- Provider-free fixtures cover API preference, anonymous UI handoff, direct
  guided-result consumption, login-required fail-closed behavior, separate
  authentication/capability profiles, optional live/portability evidence,
  package review, trusted dry-run handoff, and every tamper/private-input
  rejection introduced by A03/P01.
- A03 regression fixtures include private-root escape, explicit guided-source
  deduplication, stale/unknown/trailing input, secret/session-shaped content,
  literal guided text/select values, decision mismatch, and replay-bound cases.
- Agent/report/scorecard evidence proves iCoT never launches a browser, reads
  credential environment variables, contacts the target, or writes while
  planning; generated packages contain no raw capture, rich evidence, page
  value, credential, cookie, storage state, or live session.
- Default gates use synthetic Browsertools records and fake engines. Any
  installed Chromium/Firefox/WebKit or headed-authentication proof remains a
  separate explicit loopback-only opt-in with an honest skipped result when the
  pinned browser components are absent.
- The final cross-repo release gate covers OpenUdon, Browsertools, UWS, Udon,
  and Browserdriver without importing private runtime code or widening public
  browser semantics.

Status: implemented, reviewed, and verified in OpenUdon commit `eda602a`; see
[status-E01.md](status-E01.md).

### E02 Authenticated Authoring And Trusted Replay Qualification

**Goal.** Turn the completed authenticated authoring/context/replay milestones
into one deterministic cross-repository release-evidence matrix and operator
contract.

Dependencies: OpenUdon A04, Browsertools A03/E05/P04, UWS 1.8,
Browserdriver M03, and Udon M29.

Acceptance:

- The fixed provider-free matrix runs exact named evidence for OpenUdon's live
  client/result validation and UWS 1.7/1.8 selection, Browsertools'
  author-session/profile generation, UWS context schemas/dispatch/backward
  compatibility, Udon v2/v3 replay selection, and Browserdriver v2/v3 context
  enforcement.
- Required gates remain browser-free, network-free, target-free,
  credential-free, and subprocess-output-free. The report retains only closed
  gate results and exact clean repository revisions.
- Installed Chromium/Firefox/WebKit stays an explicit loopback-only opt-in.
  `--headed-auth` adds both authentication and same-context redirect-login
  author-session loopbacks and records an honest skip when pinned Chromium is
  unavailable.
- Browserdriver npm test and audit, full workspace and standalone OpenUdon
  tests/vet, repository boundaries, strict docs, matrix generation/verification,
  and diff checks pass. Real-site proof remains manual, non-production, and
  operator-authorized.
- Local qualification binds exact clean sibling commits. Published dependency
  pins are updated only after those commits are available upstream; no local
  replace directive or unreachable pseudo-version enters a release.

Status: implemented, reviewed, and verified in OpenUdon commit `ea81570`; see
[status-E02.md](status-E02.md).

### E03 Exact Authenticated Authoring Seam Qualification

**Goal.** Prove the exact artifacts and discriminator pairs crossing the
Browsertools/OpenUdon/Udon/Browserdriver seams and close the strict-protocol
findings that isolated repository suites did not detect.

Dependencies: OpenUdon A04/E02, Browsertools M22, UWS 1.8,
Browserdriver M04, and Udon M30.

Acceptance:

- Reduced observations carry an exact portable context inventory. OpenUdon
  rejects unreviewed origins, missing/changed contexts, unknown planner
  contexts, and unsafe raw labels before human display or model disclosure.
- Browsertools' first state and final envelope carry exactly the requested
  finite bounds; weakened or changed authority fails closed.
- Browsertools review decisions accept the producer's bounded UWS
  discriminator vocabulary, including dots and hyphens, without broadening
  fixed runtime diagnostics.
- A real Browsertools envelope crosses strict OpenUdon validation and atomic
  staging. Its exact authentication 1.1/browser 1.5 pair crosses Udon's
  v3 named session and capability output validation; contextual browser 1.6
  behavior remains covered.
- Public module pins name the coordinated feature commits, feature assertions
  never skip for an older schema archive, and the release matrix requires both
  seam-test names.

Status: complete; see [status-E03.md](status-E03.md).

### E04 Complementary Real-Browser Scenario Evaluation

**Goal.** Add real cases that exercise the complete portable Playwright-browser
workflow while keeping deterministic release qualification separate from
external-site drift detection.

Dependencies: OpenUdon A06/E03, Browsertools A04/A05, UWS 1.9/browser 1.7,
Browserdriver M05/M06/M07, and Udon M32/M33.

Acceptance:

- A required 21-case local-random-port suite runs real Browsertools
  author-session v2, strict OpenUdon profile reconstruction/staging, UWS
  synthesis, Udon v3 lowering, and Browserdriver v3 replay. It covers all eight
  MFA kinds, main/popup/frame contexts, both locator modes, zero/16/17 output
  bounds, every scalar/presence type, and closed negative cases.
- A separate four-site suite requires explicit `--allow-network`, uses only
  fixed anonymous read-only HTTPS targets and exact-one accessibility probes,
  and treats upstream drift as informational rather than a release failure.
- Manifests, dependency revisions, toolchains, report fields, failure classes,
  quarantines, attempts, and output paths are strictly bounded. Reports and
  digest sidecars contain no credentials, page content, browser state, or
  subprocess output. Duplicate JSON keys fail before decoding at any nesting
  depth, even when two spellings decode to the same key.
- The release workflow requires loopback readiness and uploads value-free
  evidence. Weekly/manual public automation is credential-free and cannot
  acquire network authority without the explicit CLI flag. Both Ubuntu jobs
  provision sandbox-compatible user namespaces and retain Chromium's sandbox.
- The clean coordinated matrix passes 21/21 loopback and 4/4 public cases at
  the exact published dependency revisions.

Status: implemented in OpenUdon commit
`0ea9f3eff281cffbcd699b2b62905090b51d5e28` and review-hardened in
`0a14a9fad7c86318f5d23028496963b4cfe01dcf`; see
[status-E04.md](status-E04.md).

### E05 Realistic Playwright-Browser Journey Qualification

**Goal.** Add realistic local application workflows to the existing portable
Playwright-browser qualification so the release gate proves ordinary read and
write journeys, not only contract-focused loopbacks.

Dependencies: E04, Browsertools guided-authoring v1/browser 1.5, UWS 1.8,
Browserdriver v3, and Udon v3.

Acceptance:

- Eight strict local manifests cover search/filter, pagination across ordered
  operations, accessibility/JSON-LD/microdata/CSS extraction, approved update,
  missing approval, ambiguous mutation locator, four parameter-contract
  failures, and fresh execution-local sessions.
- Every case generates normalized Browsertools evidence and a reviewed
  `browsertools.guided-authoring.v1` bundle, feeds it through OpenUdon's strict
  importer, and stages only the canonical browser 1.5 profile. Private
  evidence, decisions, review, spec, page content, and browser state never
  enter UWS or report evidence.
- OpenUdon synthesizes ordered parameterized browser operations in UWS 1.8.
  External Udon and Browserdriver v3 replay in real headless Chromium against
  a random-port local application, with exact output, operation approval,
  mutation count, record state, and lifecycle assertions.
- The closed `openudon.browser-journey.v1` manifest and
  `openudon.browser-journey-eval.v1` report contracts use the E04 compatibility
  lock and SHA-256 sidecars. Journey execution rejects external-network
  authority and does not require Browsertools' headed Playwright runtime.
- `make browser-scenario-journey`, `make release-saas-check`, and the tag
  release workflow require readiness and an 8/8 pass; the public canary remains
  informational and explicit-network.

Status: implemented and verified in OpenUdon commit
`5a825a41b298b4cb589756adbdef812573fc7522`; see
[status-E05.md](status-E05.md).

### P02 Trusted Execution And Package Integrity

**Goal.** Replace executable v1 handoffs with a TOCTOU-resistant v2 trusted
boundary and bind every run artifact to one unique execution identity.

Dependencies: P01 and the existing approval/package digest boundary.

Acceptance:

- `apitools.review-handoff.v2` digests every input and its cleared-field
  canonical self; package hashing still covers the final manifest bytes.
- `openudon.executor-run.v2` and `openudon.run-evidence.v2` bind run, config,
  approval, handoff, package, executor report, and async evidence. v1 cannot
  execute or be archived as v2 evidence.
- The external runner receives config path/digest plus approval and rebuilds
  all current validated state before a byte-identical canonical config can run.
- Typed executor invocations allow only declared credentials and the minimal
  launcher environment. Optional PKCS#8/PKIX Ed25519 signatures support both
  embedded integrity and separately pinned operator identity.

Status: complete; see [status-P02.md](status-P02.md).

### A12 Authoring, Provider, And UI Safety

**Goal.** Close secret-detection, provider transport, UI bootstrap,
remote-source, clone, security-selection, and HCL diagnostic gaps without
widening the single-operator product.

Dependencies: A11 implementation and P02 executable-wire decisions.

Acceptance:

- Artifact and LLM mappings share a fail-closed credential classifier while
  documented symbolic references remain valid; Gemini uses a header-only key
  and all provider bodies are bounded.
- The opened browser argv contains no capability secret. A terminal-only
  expiring, single-use, throttled Crockford code installs the scoped cookie.
- Remote HTTP validates redirects and all DNS answers, pins the dialed IP, and
  rejects unenforceable transports. One source-directory list covers every API
  and browser family across CLI, engine/UI, discovery, and seed copying.
- Draft state clones deeply, security selection uses a stable fingerprint, and
  strict HCL errors map back to the original source.

Status: complete; see [status-A12.md](status-A12.md).

### A13 iCoT UI Review Remediation

**Goal.** Remove the verified correctness and daily-use gaps in the Phase C
shell while preserving exact-revision transactions, exact-byte workspace
ownership, and the CLI-first local trust boundary.

Dependencies: A10-A12, M70/M73 shared interview contracts, and M74's v0.2
closure baseline. A13 owns the OpenUdon engine/UI adapter, embedded shell, and
operator documentation; it does not change public UWS semantics or add LLM or
executor authority. Its browser-test additions extend A11; E09 later supplied
the mandatory sandboxed release evidence for A11.5.

Acceptance:

- An accepted answer changes the authoritative targeted state; invalid or
  ineffective answers reject atomically with a question identity the shell can
  map to an accessible field error.
- Engine-owned presentation metadata drives closed choices, syntax guidance,
  and a visible structured deferral affordance; JavaScript does not infer
  workflow policy from prompts or slot names.
- An eligible settled decision can be revised through a separately explicit,
  exact-revision mutation whose refresh, autosave, drift, and failure behavior
  matches frontier rounds.
- A consumed bootstrap code has a bounded operator recovery path;
  recommendation fill never silently overwrites input; candidate, source, and
  decision evidence needed for approval is visible through safe text-only UI.
- Conditional HTTP behavior is complete, volatile browser-only state is named
  honestly, and exact-byte polling is weakened only if measurement and tests
  prove an equally safe optimization.
- Focused, browser, full, race, documentation, and repository gates pass, and
  iterative review finds no unresolved P1 or P2 in the final change set.

Status: complete; see [status-A13.md](status-A13.md). The later E09
qualification closes A11.5 with the required sandboxed Chromium pass.

### E06 Honest And Reproducible Evidence

**Goal.** Ensure release evidence never claims execution that did not happen
and remains portable and byte-stable across workspaces.

Dependencies: P02 and A12.

Acceptance:

- All-skipped browser suites report `not_run`, which cannot satisfy a passing
  verifier or readiness release target.
- Probes, builds, and scenarios use fixed deadlines and terminate complete
  process trees on Unix and Windows.
- Persisted paths are package-relative, eval workspaces self-clean unless an
  explicit archive is retained, and map-derived output is sorted.
- Slack evals use minimal operation-specific source fixtures and regenerated
  deterministic expected artifacts.

Status: complete; see [status-E06.md](status-E06.md).

### M74 v0.2 Consolidation And Release Closure

**Goal.** Consolidate shared implementation policy, remove unreachable and
duplicated code/data, document v0.2, and prepare—but do not tag—the release
tree.

Dependencies: P02, A12, and E06.

Acceptance:

- Atomic writes durably sync/rename/clean; evidence reads, strict JSON, digest
  sidecars, and complete Git IDs use shared helpers with fault tests.
- Build info, UWS validation, remediation, and catalog selection leave the CLI
  package; unused APIs and embedded schema snapshots are gone; pinned
  `deadcode v0.47.0 -test ./...` is silent.
- iCoT, progressive elicitation, and OpenAPI quality are split by
  responsibility with each former grab-bag file below 1,000 lines.
- Full standalone/race/vet/release/docs/boundary/deadcode/cross-build gates are
  recorded. E09 later closes A11.5 with sandbox-required Chromium; a
  sandbox-disabled run was never substituted.

Status: implementation complete; see [status-M74.md](status-M74.md). No v0.2.0
tag or publication is part of this milestone.

### P03 Trusted Browser Execution

**Goal.** Make reviewed browser workflows executable through the ordinary
digest-bound `openudon run` path without moving runtime semantics or secrets
into OpenUdon.

Dependencies: P02, A06, Udon browser-driver v3, and Browserdriver v3.

Acceptance:

- Browser authentication credential bindings enter the handoff inventory and
  exact declared executor environment.
- Config/evidence bind the package-derived protocol, canonical credential and
  external-session mappings, driver launcher fields, and exact reviewed
  operation/authentication approvals.
- Local and Docker Udon argv receive the complete browser CLI surface while
  unrelated, proxy, cloud, and SSH-agent variables remain excluded.
- An external runner re-derives the browser contract from current package
  bytes and rejects forged direct-runner configs.

Status: complete; see [status-P03.md](status-P03.md).

### A14 Browser Authoring And Synthesis Hardening

**Goal.** Close the browser authoring, import, lowering, quality, registry,
and process-boundary findings without expanding browser authority.

Dependencies: Browsertools M25, A06, A13, and P03.

Acceptance:

- Live frame names use canonical reduction; approval fields and origins are
  exact reviewed values; MFA subsets remain compatible and human-only.
- The live executable is privately stabilized before launch and the complete
  process group is terminated on cancellation without charging human idle time
  to an arbitrary global wall clock.
- Browser-family profile reads and discriminators fail closed; intent lowering
  validates bounded authentication fields, symbolic bindings, exact approvals,
  generic outputs, and all nested structural steps.
- One conservative document-order analyzer owns authentication-session
  readiness for nested workflows; branch-local authentication is never
  misrepresented as guaranteed.
- Registry diagnostics are control/secret-safe, equal IDs from distinct
  registries remain visible with collision-safe targets, and create-only
  staging is durable and race-safe.

Status: complete; see [status-A14.md](status-A14.md).

### E07 Authentication-Real Browser Evidence

**Goal.** Ensure browser release evidence proves authentication behavior and
the exact reviewed component state, not merely navigation-shaped replay.

Dependencies: Browsertools M25, A14, and E06.

Acceptance:

- Loopback goal pages require a random session cookie and the fixture verifies
  password plus every MFA challenge server-side before a pass is possible.
- The v2 compatibility lock rejects dirty or revision-mismatched pinned
  siblings and binds actual Playwright and Chromium versions.
- Browser integration evaluation reuses the repository-state validator and
  Playwright contract; scenario preparation compares the actual installed
  Playwright and Chromium pair.
- Failure classification is structured, journey HTTP hardening is consistent,
  loopback proxy inheritance is absent, and fixture-derived expectations do
  not duplicate markup knowledge.

Status: complete; see [status-E07.md](status-E07.md).

### M75 Browser Review Remediation Closure

**Goal.** Consolidate shared safety helpers, pin the corrected producer, and
close the browser/UI reviews with reproducible documentation and gates.

Dependencies: Browsertools M25, P03, A14, and E07.

Acceptance:

- OpenUdon pins the reviewed Browsertools M25 pseudo-version and scenario lock
  while preserving clean public-module builds without local replacements.
- Atomic create-only writes, repository compatibility validation, nested
  browser workflow analysis, and persisted browser-evidence validation have
  shared focused tests.
- README, operator/safety/eval docs, memory, statuses, and evolution v28 agree
  on the normal browser execution path and evidence claims.
- Full tests, vet, project checks, strict docs, boundary/doc-memory checks,
  formatting, dead-code, cross-build, and browser integration close. E09 later
  closes the then-pending A11.5 sandbox proof. No release tag is created.

Status: complete; see [status-M75.md](status-M75.md). E09 later closes A11.5.

### M76 Browser Execution Regression Closure

**Goal.** Remove the post-M75 browser execution and process-containment
regressions without widening OpenUdon's runtime or authoring authority.

Dependencies: P03, A14, and M75.

Acceptance:

- SIGINT/SIGTERM cancellation reaches the isolated Browsertools process group,
  and failed sessions terminate descendants even after the leader exits.
- Interactive stdout is drained before `Cmd.Wait` closes the pipe, with focused
  process tests for final buffered output and reaped-leader descendants.
- Docker browser runs mount the validated host driver read-only and pass a
  container-visible path; API-only active plans ignore retained fallback
  profiles.
- Registry collision renaming rebuilds document and operation paths and
  rechecks generated target names until unique.
- Focused, race, full, vet, project, documentation, formatting, dead-code, and
  cross-build gates pass; iterative review leaves no unresolved P1 or P2.

Status: complete; see [status-M76.md](status-M76.md). E09 later closes A11.5.

### A15 Engine Acquisition Lifecycle

**Goal.** Make journey selection, API-source intake, and reviewed browser
capture explicit engine-owned optimistic mutations.

Dependencies: A14, Apitools source discovery, Browsertools A06, and the A09
workspace transaction.

Acceptance:

- Five typed journey starters plus the operator goal are persisted as human
  decision evidence; pre-starter v2 sessions remain valid.
- Upload or browser capture, but not ordinary authoring, requires an absolute,
  non-symlink, mode-`0700` private root outside the example. The API inbox is
  capped at 20 MiB and admits exactly one secret-free, unambiguous Apitools
  source candidate.
- Upload reports the canonical target before any workspace write. Explicit
  source staging and UI-owned removal are revision-protected, atomic,
  collision-safe, digest-bound, and refresh discovery in the same mutation;
  drift and non-UI removal fail closed.
- Browser staging independently reconstructs and validates the reviewed
  authentication/capability pair, creates collision-free canonical targets,
  supports multiple profiles without overwrite, and leaves final flow/action/
  session/approval selection to ordinary workflow approval.
- Interactive, complete-session, agent, progressive, restart, and engine/UI
  refreshes use one source-policy routine for inactive/ambiguous/truncated
  assessment, registry triggers and selected-digest revalidation, source-plan
  synchronization, and verification attachment. Every browser step has its
  own source/action/session/approval frontier identity.
- Safe authenticated-authoring review is an append-only, 128-entry v3
  collection. The next successful stage migrates one valid v2 singleton
  deterministically; malformed, changed, duplicate, or colliding review state
  writes nothing.
- Focused tests prove secret/ambiguity rejection, upload cleanup, restart,
  removal drift, v2 migration, genuine second-profile staging, review-file
  workspace drift, and failed/canceled/expired/malformed capture cleanup.

Status: implementation complete; see [status-A15.md](status-A15.md).

### A16 Unified iCoT UI And Package Handoff

**Goal.** Make iCoT UI the primary interactive API/browser authoring surface
while keeping Playwright and trusted execution behind separate process and
approval boundaries.

Dependencies: A15, Browsertools A06, P02/P03, and A13.

Acceptance:

- One distributed `icot` privately stabilizes and re-executes a hidden
  Browsertools author-session/doctor worker. UI and bundled terminal mode share
  one typed coordinator; only terminal mode retains an expert external-binary
  override. The engine and HTTP server do not initialize Playwright.
- Experimental `openudon.icot-ui-api.v3` serves no v1/v2 routes, preserves the
  access-code/cookie/bearer and loopback Host/Origin boundary, keeps authoring
  and capture revisions separate, consumes each capture response revision,
  binds the complete ETag, serves snapshots during readiness/capture, and
  permits only one active capture.
- Chromium readiness uses Browsertools' typed doctor with a 30-second bound.
  Capture retains the ten-minute browser-active limit, 30-minute operator-idle
  cancellation, two-hour absolute ceiling, process-group cancellation, stdout
  draining, and descendant termination. Canceling remains an active capture
  until teardown is confirmed; a process-containment timeout fails closed and
  requires restart before further authoring or capture.
- Bundled, expert external, UI, and scenario capture use one controller. A
  non-serializable parent attestation binds the exact ordered actions,
  checkpoints, approvals, observations, contexts, outputs, authentication
  proof, diagnostics, and approved-origin ledger before staging. Unsafe
  configured or page-derived paths fail before terminal/HTTP/model disclosure.
- The accessible shell exposes journey/API helpers, safe doctor state, typed
  reduced capture states, exact origin/action/POST approvals, Chromium-only
  credential guidance, Browsertools-reported MFA choices, bounded output
  review, provenance, symbolic credential/session maps, workflow graph, and
  recovery guidance. Sensitive values and private worker fields have no HTTP
  representation.
- Authoring approval writes the reviewed authoring artifacts and enters
  `authored`. A separately confirmed two-minute deterministic build plus
  non-writing assessment of the exact resulting bytes either gives check-
  specific remediation and requires revision-protected resume/reapproval, or
  freezes a passing handoff.
- Handoff inspection is a bounded closed allowlist with quality, side effects,
  symbolic credentials/sessions, approvals, package/handoff digests, and exact
  approval-template argv. Policy facts and hashes bind one stable handoff
  generation, the complete digest-bound package stays unchanged across
  assessment, and later artifact drift invalidates the frozen handoff.
  No registration, approval-generation, credential, run, or execution route
  exists.

Status: complete; the hardening follow-up is published and passes coordinated
plus standalone verification; see
[status-A16.md](status-A16.md).

### E08 Unified UI Real-Browser Qualification

**Goal.** Prove the complete access-code-to-handoff lifecycle in real
sandboxed Chromium without leaking operator-entered values.

Dependencies: A15, A16, Browsertools A06, and A11's listener harness.

Acceptance:

- Provider-free unit/race coverage drives reduced observations, exact
  approvals, credential/MFA checkpoints, bounded output/completion review,
  cancellation, idle/absolute timeouts, malformed result/protocol, crash,
  stdout/process-tree cleanup, independent revision consumption, and one-
  capture enforcement.
- Adversarial coverage rejects unsafe disclosure paths, fabricated or reordered
  traces, changed authentication/output/origin facts, stale approvals,
  context/diagnostic overflow, missing attestation, and credential answers that
  would otherwise mutate draft bytes.
- API upload/removal, append-only v3 review, valid-v2 migration, true
  multi-profile staging, package failure resume/reapproval, current-byte
  assessment, frozen-package drift invalidation, artifact allowlisting,
  terminal compatibility, and absence of approval/run routes are covered.
- A real loopback journey performs access-code exchange, starter selection,
  login/MFA entry only in Chromium, capture review/stage, continued interview,
  authoring approval, separately confirmed package build, and passing handoff.
- The same journey asserts credential and challenge values are absent from
  HTTP requests, DOM, logs, sessions, profiles, reviews, packages, child argv,
  and child environment.
- Full tests/vet/docs/boundaries/release gates, sandboxed UI Chromium, deadcode,
  formatting, and CGO-disabled Linux/Windows/macOS builds pass against the
  published Browsertools A06 pin.

Status: original qualification complete; the hardened publication, cold
standalone, unit/race/integration, and headless-journey expansion are green.
Sandbox-required Chromium must rerun on a host with sandbox-compatible user
namespaces. See
[status-E08.md](status-E08.md).
E09 retains the prior published-pin evidence.

### P04 Immutable Trusted Execution And Containment Closure

**Goal.** Bind trusted execution to one validated package byte generation and
close process/environment containment gaps.

Dependencies: P02/P03, Udon M35, and Browserdriver M08.

Acceptance:

- Every required package input is read once; declared digests, quality,
  package/handoff digests, plan/intent/review/profile interpretation, browser
  protocol, credentials, sessions, approvals, and run config use that snapshot.
- Staging rehashes current files and rejects drift before execution.
- Normal exit and cancellation sweep descendants; Linux identity-tracks and
  verifies detached children without PID-reuse kills, with documented Unix and
  Windows fallbacks.
- Docker forwards only declared credential/session values and uses container-
  owned driver environment defaults; outer-runner Udon overrides and one
  canonical credential-name derivation are preserved.

Status: complete; see [status-P04.md](status-P04.md). Final release evidence
closed under E09.

### A17 Acquisition, Synthesis, And UI Remediation Closure

**Goal.** Close remaining workspace drift, legacy migration, stable-read,
nested-source, and UI response-state findings.

Dependencies: A15/A16 and published Browsertools A06.

Acceptance:

- Capture staging and source removal observe before semantic reads and repeat
  authority checks inside replacement; browser staging always has a private
  root.
- Legacy review migration accepts only exact digest-derived IDs and complete
  current safe-evidence invariants.
- iCoT executable copying and packaged profile parsing bind one stable content
  generation; nested quality/lowering share one effective-source traversal and
  conservative session analysis includes `loop`.
- UI doctor state is path-free before storage/ETag/serialization, failed
  initial revision generation rolls preflight back, and JSON depth is capped at
  64.

Status: complete; see [status-A17.md](status-A17.md).

### A18 Secret-Safe Browser Registration Intent And Packaging

**Goal.** Add an inert, dependency-closed OpenUdon producer boundary for the
published browser-registration profile and call without implying runtime
support or contacting a target.

Dependencies: UWS `v0.0.0-20260825173657-a539de17eea3`, Browsertools
`v0.0.0-20260825175014-c37744a1a546`, A02, P04, and A17.

Acceptance:

- An explicit `browser_registration` intent names one reviewed package-local
  profile, flow, complete symbolic credential map, exact approval, fixed
  duplicate/ambiguity policy, cleanup disposition, and bounded timeout; it has
  no browser session.
- Synthesis lowers the step to `uws.browser-registration-call.1.0`, selects a
  current compatible UWS core version, and records the account-creation side
  effect without exposing an account identifier or credential value.
- Package quality and handoff require the exact profile, its verified
  `browsertools.registration-review.v1` bundle, and strict OpenUdon review
  inventory with matching digest, origins, flow slots, expiry, approval, and
  cleanup policy.
- Approval-template and trusted-runner dry-run validate the immutable package
  using symbolic credential environment names only. Every executor-building
  path rejects registration before process invocation until a compatible Udon
  and Browserdriver contract is published and pinned.
- Synthetic tests, secret/PII adversarial tests, full workspace and standalone
  module tests/vet/race, repository/documentation gates, and diff review pass
  without browser or network access.

Status: complete; see [status-A18.md](status-A18.md).

### E09 Post-Remediation Release Evidence Closure

**Goal.** Make expected failures typed, interactive MFA server-observable, and
release evidence clean-root, standalone, and sandbox-mandatory.

Dependencies: P04, A17, Browserdriver M08, Udon M35, Browsertools A06, and E08.

Acceptance:

- Only strict `udon.execution-report.v2` codes classify runtime failures;
  `unclassified` records malformed/unrelated failure but can never satisfy a
  negative scenario.
- Both browser evaluators reject dirty OpenUdon and sibling roots while
  explicitly ignoring generated `site/`; public evidence never classifies
  stderr substrings.
- Approval MFA waits for Udon's exact prompt, latches exactly one pending
  observed server session, and rejects direct challenge POST bypass; loopback
  runtime state and routing are race-free.
- Required UI qualification asserts/logs Chromium sandbox use and rejects the
  disable override; an unsandboxed target is diagnostic only. Normal and
  release CI build iCoT standalone early.
- Published Browserdriver/Udon/Browsertools commits, the Browsertools module
  pin, compatibility lock, standalone gates, clean-root matrices, real
  sandboxed Chromium, docs, race/deadcode, and cross-platform builds all pass
  before downstream publication.

Status: complete; see [status-E09.md](status-E09.md).

### M77 Browser-Authoring Transaction Contract And Documentation

**Goal.** Establish the public, value-free transaction semantics shared by
browser authentication, capability, and registration authoring before any
producer or consumer implementation advances.

BAP, BCP, and BRP denote the existing browser-authentication, browser
capability, and browser-registration profile families. `BxP` is only shorthand
for this set; it does not allocate a new UWS discriminator.

Dependencies: existing OpenUdon commits `07f5d48` and `854e146`, published
Browsertools `5aaeb45`, and published UWS `895aa45`.

Acceptance:

- One strict transaction model binds kind, lifecycle, exact source/review
  digests, preparation/promotion state, private-result provenance, and typed
  failures without values or private paths.
- Public documentation distinguishes BAP+BCP named-session composition from
  BRP no-session/no-submit authoring and preserves unsupported registration
  runtime failure before executor invocation.
- Product, architecture, tech-stack, milestone, API/CLI compatibility, and
  security guidance agree; UWS remains unchanged unless an actual missing
  public semantic contract blocks progress.
- Evolution prompt v31 was created when M77 started; result v31 now records
  the complete sequence through offline W01.
- Contract, compatibility, documentation, and bounded review-fix gates pass,
  and every named downstream status is reconciled.

Status: complete at OpenUdon `96eb4b6e894c47900e7f95617a5146be6cdff454`;
see [status-M77.md](status-M77.md).

### A19 Private Browser-Profile Candidate Lifecycle

**Goal.** Adopt the published Browsertools producer result as private,
reviewed BAP/BCP/BRP transaction candidates without promoting its envelope or
browser state into the workspace.

Dependencies: M77 and published Browsertools
`39e32c1d6f601561cc5c13ec85201815ce85ab9b`.

Acceptance:

- Stable, strict private-result adoption independently rebuilds and validates
  candidates and rejects provenance, freshness, digest, path, or generation
  drift.
- Virtual source discovery exposes deterministic kind-specific candidates
  without pre-approval workspace mutation and preserves API-first preference.
- BAP+BCP compose through one execution-local named session; BRP remains
  session-free, no-submit, and non-executable outside dry-run packaging.
- Cancel/resume/review/reapproval and partial-write recovery are deterministic,
  with required focused/full/security and bounded review-fix gates passing.

Status: complete with A19.1 at OpenUdon
`b5fbc0b6c49ca4a4aeb746afd592f51942b81249`, A19.2 complete at
`3c1a5e7d7b98bcba09b1849d4b41c1429c5c0fc5`, A19.3 complete at
`21f59f22cfd1f102893b91ef542fbe3c44265e41`, A19.4 complete at
`f7d7163b743ef3f635f029bbdbd0c8d0216e0e2e`, and A19.5 at
`1fefb8e7672e542a6f6988f5e3cf6a09632c898b`; see
[status-A19.md](status-A19.md).

### P05 Prepare-Only Build And Atomic Promotion

**Goal.** Separate pure package construction from restrictive qualification
and make the final workspace change one recoverable atomic promotion.

Dependencies: M77 and A19.

Acceptance:

- Preparation is deterministic and write-free over one validated byte
  generation.
- Qualification occurs in a restrictive same-filesystem scratch boundary and
  runs all package, review, secret, handoff, and dry-run gates before promotion.
- Concurrent promotion is atomic and collision-safe; readers observe an old
  or new complete package, and selected/prior generations are preserved.
- Typed rollback and indeterminate recovery reject blind retries and pass
  fault-injection, compatibility, release, and bounded review-fix gates.

Status: complete with P05.1 at OpenUdon
`21a32edb982da399cc6e05fd97d6a1f350aca4d6`, P05.2 complete at
`dcb5686ffe175b1dfe62f1e9735af288eaed1828`, P05.3 complete at
`319babd8033a871143f665c3c79358bf4ef23129`, P05.4 complete at
`e3c7497409d1b2d2c9c06e344120b99e6a46d2e4`, and P05.5 at
`2121f06d6eab173012ab9d2a0f797ea45cece617`; see [status-P05.md](status-P05.md).

### A20 Unified Browser Transaction Engine, UI, And Terminal UX

**Goal.** Put BAP+BCP and BRP authoring, review, preparation, promotion, and
recovery behind one engine used consistently by the local UI and terminal.

Dependencies: M77, A19, P05, and published Browsertools
`39e32c1d6f601561cc5c13ec85201815ce85ab9b`.

Acceptance:

- One driver-agnostic engine owns typed transaction snapshots, revisions,
  approvals, transitions, cancellation, and recovery.
- Experimental UI API v4 exposes only value-free transaction state, retires
  the replaced v3 surface, preserves local security bounds, and has no runtime
  execution route.
- The embedded shell provides an accessible, conflict-aware transaction review
  and separate prepare/promotion decisions for BAP+BCP and BRP.
- Terminal behavior remains compatible through the shared engine, and focused,
  accessibility, full, release, loopback, and bounded review-fix gates pass.

Status: complete; see
[status-A20.md](status-A20.md).

### E10 Cross-Package Browser Transaction Qualification

**Goal.** Prove the complete public producer/consumer/package/UI seams with
value-free evidence before private W8M reconciliation.

Dependencies: M77, published Browsertools A08
`39e32c1d6f601561cc5c13ec85201815ce85ab9b`, A19, P05, A20, and published UWS
`895aa45`.

Acceptance:

- A deterministic report binds exact commits and artifact digests while
  excluding all values, page/request data, session material, raw output, and
  private paths.
- A real Browsertools-produced BAP+BCP pair passes same-session loopback
  authoring through sandbox-required replay.
- A real Browsertools-produced BRP passes a zero-POST loopback producer path,
  OpenUdon prepare/promotion, and dry-run handoff while non-dry execution fails
  before executor invocation.
- Adversarial, concurrency, rollback, recovery, leak, standalone, release,
  cross-build, UI, browser, and bounded review-fix gates all pass.
- Browsertools A08 is independently resolvable, OpenUdon stays unpushed through
  W01 closeout, and evolution results are written only after W01 passes.

Status: complete at OpenUdon
`bb69c5a530eac303646e49a37455ce1bf19b3f57`, subsequently published with the
qualification-state follow-up at
`42767a160dc88ac18ac5d624ad9e17151ba50d77`; see
[status-E10.md](status-E10.md).

### M78 Trusted Browser Registration Handoff

**Goal.** Accept Browsertools registration-authoring v2 through an additive
transaction v2 and permit non-dry registration only through the exact external
Udon v4, Browserdriver v4, execution-report v3 contract plus a private
digest-bound attestation.

Dependencies: Browsertools M27/A09, Browserdriver M09, Udon M36, unchanged
published UWS registration 1.0, and OpenUdon M77/A19/P05/A20/E10.

Acceptance: transaction v1 and every BAP path remain unchanged; transaction v2
is registration-v2-only; the owner-readable untracked attestation carries no
account value; missing, expired, drifted, previously attempted, legacy, or
incomplete configuration fails before executor invocation; exact v4/report-v3
handoff, submit approval, local/Docker/outer-runner revalidation, full gates,
and two reviews pass without publication.

### A21 Guided iCoT Browser Registration Authoring

**Goal.** Add one accessible, revision-protected BRP wizard over Browsertools
registration-authoring v2 while preserving existing BAP/BCP authoring.

Dependencies: M78 and Browsertools M27/A09.

Acceptance: authenticated API v4 start/command/cancel, one isolated worker,
guided metadata/slots/macro/locator/effect/success/control/cleanup/query review,
no credential or verification fields, explicit canonical confirmation, clean
teardown before adoption, existing package lifecycle integration, real
loopback UI/accessibility/process tests, legacy regression, and two reviews
pass without publication or real target contact.

### E11 Browser Registration Runtime Qualification

**Goal.** Prove the complete general BRP authoring-to-runtime path and every
preserved BAP behavior before private W8M authoring begins.

Dependencies: M78, A21, Browsertools M27/A09, Browserdriver M09, Udon M36, and
unchanged published UWS registration 1.0.

Acceptance: exact clean locks, one synthetic loopback approved POST, all
attestation/approval/checkpoint/indeterminate/retry/origin/teardown/no-session
cases, BAP authentication/context/MFA/scalar/UI regressions, full standalone/
release/audit/secret gates, canonical value-free evidence, and two reviews
pass. Only then are coordinated evolution results v16/v6/v14/v32 written. No
publication or real target/account authority is included.

Status: complete locally at OpenUdon
`f75d9634508ee81f74932cb2d3c219ef0a51c19e`; coordinated results are local at
Browserdriver `a269e94`, Udon `1d15191`, and Tofu `95752f6`; see
[status-E11.md](status-E11.md). At E11's close, W8M W02 remained gated and
unstarted.

### A22 Operator-Authored Content Trust

**Goal.** Let an operator author reviewable UWS content-trust declarations in
workflow intent and synthesize them without changing legacy packages.

Dependencies: published UWS `9e676eaa469e` and published Browsertools
`75fd5c3ab81f`.

Acceptance: intent represents source-description, operation-output, trigger,
and workflow-input/default declarations with strict levels and deterministic
HCL; synthesis maps them to the generated UWS identifiers; documents with a
non-empty declaration use UWS 1.9.1, while otherwise unchanged browser 1.7
packages remain byte-compatible UWS 1.9.0; malformed or unresolved declarations
fail authoring/synthesis before packaging; focused, full, standalone, vet, and
diff gates pass.

Status: complete at OpenUdon `51de359` and subsequently published through
`2c99fde`; see [status-A22.md](status-A22.md).

### P06 Advisory Content-Trust Review Evidence

**Goal.** Run explicit UWS/Browsertools content-trust analysis during package
assessment and surface deterministic findings for review without converting
them into execution authorization.

Dependencies: A22, published UWS `9e676eaa469e`, and published Browsertools
`75fd5c3ab81f`.

Acceptance: contained browser profiles feed Browsertools' resolver; every
analyzer finding appears as a warning quality check and value-free review
evidence in stable order; resolver/load failures are visible but advisory;
ordinary UWS validation, passing quality status, trusted-runner approval, plan,
and execution behavior are unchanged; focused, full, standalone, vet, strict
docs, and diff gates pass.

Status: complete at OpenUdon `3c9d0d9` and subsequently published through
`2c99fde`; see [status-P06.md](status-P06.md).

### E12 Content-Trust Compatibility Qualification

**Goal.** Prove the additive UWS 1.9.1 authoring and advisory-analysis contract
over the exact published upstream commits and legacy OpenUdon behavior.

Dependencies: A22, P06, and published Udon `207e7f1` (M37).

Acceptance: offline cases cover mail-to-LLM data, untrusted instruction,
model-output-to-side-effect authority, constrained control flow, trigger
payload, unknown extension, resolver failure/conflict, and unchanged legacy
packages; tagged Udon compatibility uses the public M37 entry point without
adding a production dependency; report/review evidence contains paths and fixed
messages but no runtime values; complete standalone, race, vet, release,
documentation, secret, and bounded review gates pass. No push, tag, release,
live target, or W8M action occurs.

Status: complete at OpenUdon `cc378be` and published with the clean-checkout CI
follow-up `2c99fde`; see [status-E12.md](status-E12.md).

### A23 Browsertools A10 Consumer Adoption

**Goal.** Adopt the published Browsertools registration resource-policy
remediation through an exact module pin and requalify OpenUdon without changing
its authoring or runtime authority.

Dependencies: published Browsertools A10
`3107470313d447e29c5ac5912c3cc9d221d46967`.

Acceptance: `go.mod` and `go.sum` name the exact published pseudo-version; the
content-trust dependency qualification expects the same version; full offline
tests, vet, `make check`, sibling/API boundary checks, documentation-memory
validation, and diff checks pass; no OpenUdon public wire, Browsertools policy,
target allowlist, browser session, account action, or runtime execution changes.

Status: complete and published at OpenUdon
`dd7437c0149903ee7af987cd2e02380735ccbc40`; see
[status-A23.md](status-A23.md).

### A24 Honest Guided Registration Profile Completion

**Goal.** Make the guided no-submit registration wizard produce only honest,
reviewable profile input when a registration form needs a contact name,
password confirmation, and a post-submit outcome that cannot be observed
without submitting.

Dependencies: A21, unchanged UWS browser-registration 1.0, and published
Browsertools A10.

Acceptance: the wizard defaults a value-free `contact_name` symbolic slot;
credential steps select declared slots and may safely reuse `password`; the
success locator is an explicit operator-reviewed declaration, not a current-
page observation, and is disclosed as deferred until runtime; submit remains
bound to one exact observed current-generation candidate; focused/full offline
gates and documentation review pass. No additional public target contact during
A24 implementation, submission, account action, execution, release, or
deployment occurs.

Status: complete and published at OpenUdon
`bf90537be965ace7885d43de53dab1f9eb2f2ab9`; see
[status-A24.md](status-A24.md).

### A25 Guided Registration Symbolic-Binding False-Positive Closure

**Goal.** Let the guided registration draft accept valid namespaced symbolic
runtime binding names without mistaking them for credential values, while
retaining fail-closed rejection of genuinely secret-shaped input.

Dependencies: published A24 and unchanged UWS and Browsertools contracts.

Acceptance: registration draft construction accepts descriptive high-entropy
portable names whether or not they repeat the declared slot, using at least two
closed-vocabulary purpose words and at most one alphanumeric namespace
component of up to 12 characters with one digit run; known credential formats,
short opaque-prefix suffix bypasses, digit-bearing or letters-only opaque
multi-token names, invalid symbols, duplicate slots, and duplicate bindings
remain rejected with a value-free binding-specific API diagnostic; focused and
full offline tests, vet, repository checks, documentation-memory validation,
JavaScript syntax, diff checks, and a bounded review pass. No public wire,
dependency, target contact, browser session, submission, candidate, transaction
adoption, runtime execution, publication, release, or deployment changes.

Status: complete and published at OpenUdon
`561fd933097abe90cdbe2b59b56e0fa33d4d41c1`, correcting intermediate revision
`e5b8a87d00ffd158dc7b71e7b5c264d320bf425f`; see
[status-A25.md](status-A25.md).

### A26 One-Attempt Registration Authoring And Terminal Failure Retention

**Goal.** Make one-session registration-authoring authority enforceable by the
iCoT server and preserve a useful closed failure class after clean teardown,
independently of chat or client activity indicators.

Dependencies: published A25 and unchanged UWS and Browsertools contracts.

Acceptance: once authenticated start preconditions pass, the server consumes
one process-local attempt immediately before worker construction; rapid,
stale-revision, active, failed, canceled, and pre-launch-failure repeats cannot
construct another worker and return one fixed conflict; additive API v4 state
exposes `attempt_consumed` and only an allowlisted `failure_code`, with unknown
input collapsed to `worker_failed`; the UI disables Launch and explains the
fresh preflight/authorization/process requirement; focused/full offline tests,
race tests, vet, repository checks, documentation-memory validation, JavaScript
syntax, diff checks, and bounded review pass. No dependency, public UWS or
Browsertools wire, target contact, browser session, submission, candidate,
transaction adoption, runtime execution, release, or deployment change.

Status: complete and published at OpenUdon
`e40f56fc71b9e336f270118dde9cc1830326b489`; downstream pin adoption remains
separate. See [status-A26.md](status-A26.md).

### A27 Registration Terminal Delivery And Global Containment Propagation

**Goal.** Make worker teardown failure unmissable and process-global across all
later browser-sensitive iCoT actions.

Dependencies: published A26 and unchanged Browsertools registration-author
wire.

Acceptance: joined `ErrTerminationTimeout` reports fixed `worker_teardown`;
the session retains one final value-free event outside its bounded stream; the
UI reconciles that outcome after stream closure; registration teardown failure
blocks registration, capture/preflight/staging, package/resume,
browser-transaction, and ordinary authoring mutations until process restart;
Linux PID/start-time records are immutable, procfs health loss fails closed,
live-leader group signaling precedes cancellation reap, post-wait cleanup never
signals a recycled numeric process group, and non-Linux fallbacks remain
unchanged; focused
and full tests, race, vet, `make check`, diff, documentation-memory, and bounded
review pass. No public wire, dependency, target, browser, submission,
candidate, or runtime authority changes.

Status: complete and published at OpenUdon
`552ee3a88a595cb25af1d089a0d3beeebba2a24f` after exact Browsertools A11
adoption; W8M pin adoption remains separately owned. See
[status-A27.md](status-A27.md).

### A28 Private Registration Discovery Inventory

Owner-authorized UI and shared-application scope follows completed A27/M83.
Keep discovery inventory, limits and owner review private, expose explicit
HTTP/control resources, and allow candidate selection to prepare the wizard.
UWS published 1.1 remains compatible; new recipes omit discovery metadata.
Require private history, stale-write and authority-isolation regressions plus
loopback UI verification. No automatic crawler, typed-input runtime adoption or
live W8M operation is part of this milestone. Permanent task/review truth lives
in [status-A28.md](status-A28.md).

## Documentation Maintenance

- Update this file when sequencing, contracts, milestones, acceptance criteria, active/parked track
  summaries, current-state dashboard content, or the status-file index change.
- Create or update the relevant `status-Mx.md` file for milestone task rows, task states, notes, and
  scoped commit tracking. Future milestone task history belongs in individual status files, not in a
  root aggregate status document.
- After each major milestone or boundary change, decide whether [tabilet/evolution/](../evolution/) needs a
  new prompt/result version.

### M79 Consolidated Browser System Engineering

Outcome: one application lifecycle, real UI/worker/package journey and one
aggregate local gate across browser authoring and trusted execution, removing
MCP orchestration from the supported engineering path.

One coordinated implementation row owns this boundary; see
[status-M79.md](status-M79.md). Required acceptance includes the existing
v2/v3/v4 execution matrices, approval and uncertain-submit no-retry boundaries,
lifecycle and isolation regressions, source/evidence verification, three fresh
complete local passes and a maximum-ten-iteration review-fix gate. No P1/P2
finding may remain at closure. Source publication and real-target operation
remain separate. Default tests are browser-free.

M79 is complete locally after three full eleven-stage passes, independent
verification of source/component digests, and bounded review iteration 5.
No in-scope P1/P2 finding remains. Publication and real-target operation retain
their separate authority boundaries; see status-M79.md.

### M80 Supervised Application Authoring And Packaging

Complete locally under the approved W8M W08 coordinated goal. The shared application
service covers BRP and authenticated capture through authoring and package
promotion over an opt-in private protocol, preserving all existing UI,
transaction, credential and execution boundaries. See status-M80.md. W08 owns
the shared review counter and downstream W09–W14 integration dependencies.

M80 closes at bounded W08 review iteration 2. W09 owns the next coordinated
W8M integration and acceptance unit; see W8M status-W09.md.

### M81 W8M Operating Integration And Acceptance

Complete locally under W8M W09 after completed M80/W08. Preserve human verification
interaction through the trusted runtime handoff and extend the generic
qualification inventory consumed by W8M's adapter and three-workflow acceptance.
See status-M81.md; W09 owns the shared defect list and bounded review counter.

Three complete W8M units pass with 117 native stages and nine workflow receipts,
independent source/runtime verification and clean W09 review iteration 6.
W10 owns publication and immutable operational adoption; real W8M execution
remains separately authorized downstream work. OpenUdon contains no target code.

### M82 Single pre-submission recovery attestation

Complete: additive private attestation v2 truthfully records one failed
pre-submission attempt and binds an independently reviewed authority/claim and
prior evidence. V1, UWS registration 1.0, run-config and execution evidence
formats remain unchanged. The operating application owns persistent one-use
consumption and independent historical proof. See [status-M82.md](status-M82.md).

Three complete consumer qualification units pass with nine native browser
passes, 117 stages and nine synthetic workflow receipts. Independent source,
runtime and native-owner verification pass; bounded review iteration 5 closes
with no remaining P1/P2 finding. Exact tested runtime bytes remain preserved
across later coordination commits.

### M83 Attestation For A Second Explicitly Authorized Recovery

Complete after M82: private attestation v3 records exactly two prior
attempts, unchanged immediate-predecessor digest links, delete_separately and
twenty-minute expiry. Keep v1/v2 semantics unchanged. The operating application
owns complete historical proof, fresh owner authority and exclusive second-claim
consumption. Require native positive/negative and trusted-handoff checks, full
module/quality verification and fresh complete consumer qualification before
adoption. See [status-M83.md](status-M83.md).

All three complete units pass with nine native browser passes, 117 stages and
nine synthetic workflow receipts. Independent source/runtime and native-owner
verification pass; all three tested runtime copies are preserved and match.
Bounded review iteration 2 closes without a P1/P2 finding. Existing v1/v2,
UWS, run-config and execution-receipt contracts remain unchanged.

### E14 Registration Foreground And Deadline Integration

Browserdriver M12 owns foreground requests and private v5 checkpoint deadlines;
Udon M40 owns earliest-bound timing, private form display and late-action refusal.
OpenUdon pins the reviewed revisions and exercises countdown visibility through
the actual typed UI-to-runtime qualification. Public BRP/UWS and private-input
identity remain unchanged. Acceptance requires owner checks, fresh affected
smoke, complete W16 acceptance v2, independent tested-byte adoption and bounded
review with no P1/P2. W8M owns live authority and consumed-attempt evidence.
See [status-E14.md](status-E14.md).


## E15.4 renewed qualification closure — September 14

OpenUdon 58d41f819e81, Udon 2fb0982660dd and Browserdriver aefdd875633b
complete the published sandbox-handoff correction and renewed W16.4i.2 gate:
39 fresh native stages, three W8M journeys, independent verification and exact
retained-byte adoption. E15.4 review iteration 4 passes; see
[status-E15.md](status-E15.md). Prior qualification and consumed evidence remain
preserved. English authoring/probing and real registration require separate
scope; no live operation is armed.


### E19.2 / W16.4i.39 current integration follow-through

Three fresh isolated initialization-file runs and the original five-file native
stage pass on Browserdriver 8b63833, with canonical proof, clean teardown and
bounded review. OpenUdon binds the tested diagnostic candidate for publication.
Historical .37e remains failed/consumed with an unresolved cause. E19.2 now
awaits publication and the newly selected complete qualification before adoption;
inherited E13/A27 work and the existing unconsumed human probe remain preserved.


### E19.2 / W16.4i.39d qualified and adopted successor

W16.4i.39d passes one fresh acceptance-v2 qualification: all 39 native stages
and three W8M synthetic journeys. Independent canonical report, source/runtime,
fixture, preservation and cleanup checks pass, followed by bounded reviews 3–5.
All 5,958 execution, five verifier and four adoption-check identities are absent
without force. Exact retained pass-one bytes are adopted; no rebuild occurs.

The selected closure is W8M c512075, OpenUdon 19e5d9c, Browserdriver 8b63833
and coordination Tofu afb8fba. Driver v9 and verification diagnostics/probe v5
are adopted; registration remains v6. The .28d runtime and every consumed
failure stay preserved. Historical initialization/authoring causes remain
unresolved; the passing run does not establish them.

The authorized .34e probe subsequently failed with verification_timeout and
is consumed. Native evidence observes a callable Turnstile API and an associated
visible frame, beyond the earlier missing-API symptom, but no usable response.
Independent verification rejects readiness and confirms zero application POSTs
and complete unforced cleanup. The provider failure cause remains unresolved.
The qualified .39d runtime stays adopted; no retry or registration follows.

E19.2 completes its implementation and integration acceptance under W8M
reviews 3–5. The earlier E18/.34d and E19/.37e failed attempts are unchanged;
their adoption prerequisite is resolved by this new successful successor.
The separate .34e live probe is now failed/consumed; W8M owns its failure closeout.
