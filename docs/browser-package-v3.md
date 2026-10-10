# Package v3 browser supplement

P10 adds an explicit browser supplement to the public `packagev3` byte API.
It consumes ordinary published Browsertools M33
`v0.0.0-20261010115646-1859f5e34367` and UWS C10
`v0.0.0-20261009220914-f01a2542410c`. The independent retained HCL codec stays
at `v0.0.0-20261008043726-b099f6803277`.

`SourceInput.Kind == "browser-profile"` admits exact native action,
authentication and registration profile bytes under `sources/browser-profile/`.
Build and Verify independently reproduce the complete shape table with
Browsertools; caller-created shapes or matching hashes alone establish no proof.
Selected digests cover complete lossless native action/flow subtrees, while
source digests separately cover every original byte, including shared contexts.
Native schema validity and runtime admission are separate checks.

Browser packages include `expected/browser.json` (`openudon.browser-supplement.v1`)
in the manifest/input digest and complete handoff inventory. The manifest's
optional `browser` reference and `browser_reviews` inventory bind that supplement
and its separately reviewed artifacts. Non-browser packages omit both fields
and preserve the existing package/assessment/handoff/plan bytes and digests.

The supplement binds the exact ordered browser leaves in one straight-line
workflow, source/selected hashes, per-leaf protocol pair, complete origins,
effects, confirmation-policy digest, symbolic credential names and declared
registration input binding. A browser call never carries credential values,
registration snapshots, browser state, local paths or driver arguments.
`BrowserBuildOptions.Permissions` makes explicit per-step reuse/save choices;
zero values select fresh contexts. Unknown step choices, registration reuse/save,
or action save permissions refuse. The host separately confirms current
credential/session authority and supplies isolated runtime admission.

`VerifiedPackage.BrowserSupplement()` returns an independent copy. The existing
execution plan adds an optional `browser` call only for browser leaves and
requires the host's implementing runtime admission before deriving their plan.
Old non-browser approvals never acquire browser authority by this addition.

These APIs perform no browser launch, external request or private-runtime import.
Kinet M56/M52 own actual containment/custody and later actual producer
interoperability qualification. Frozen synthetic M51 vectors are metadata
conformance inputs and grant no execution authority.

## Public verification subset

Public `browsertransaction` supports the unchanged transaction receipt v1–v4
records, canonical digests, strict decoding and closed lifecycle transitions.
The retained CLI consumes aliases to that public implementation; no authoring
engine, synthesis, wizard, runtime or trusted-runner package enters its closure.
All original transaction and report regression tests remain active at their
public owner paths, and the focused Make verification selector follows them.

Public `browserverify.InspectBytes` verifies explicit value-free live/portability
reports against one exact native profile without filesystem/browser access.
The retained file readers preserve their earlier bounded regular-file checks.
`DecodeCaptureReceipt` reads the unchanged `openudon.browser-capture-import.v1`
wire with transaction versions1–4. `VerifyCaptureReceipt` requires independently
retained original receipt/start/transaction identities and exact public
source/review bytes; changed/missing/expired evidence refuses. Omitting `At`
is historical byte inspection only. Current adoption requires the host's time,
issued original capture identity and separate human confirmation.

Package review artifacts remain bounded explicit reviewed inputs. Their hashes
enter the supplement and complete package input identity; possessing those
bytes never proves an actual capture or supplies new host authority. Kinet's
capture adapter uses the public receipt verifier before adopting an issued
capture. No raw private envelope, live browser state, credential or model input
is promoted into the public verification subset.

## Browser approval and Authority

The additive `openudon.browser-approval.v1` receipt is sandbox-only, independent
of existing approval v1/v2. It binds the complete browser plan/calls, owner,
agent, attempt, source/package/input/supplement identities, worker/driver,
original deadline, exact origins, symbolic credential revisions and the single
optional durable-session permission. `ReviewBrowserConfig` constructs only
review metadata; Kinet still renders and confirms its exact bytes.

Construction is acyclic: finalize the exact human receipt without an enclosing
Config/Authority hash, obtain independent host facts, construct Config with the
receipt byte hash, then compute its complete lossless canonical hash and
Authority. `DeriveBrowserAuthority` requires a verified package, native runtime
admission, independently expected current Config, exact finalized receipt,
separately confirmed plan hash and explicit host time. It reproduces every
browser call and refuses changed bytes/identities, expired deadlines, stale
sessions and widened origins/credentials/reuse/save permissions. Existing
`CheckExecutionApproval` refuses any browser leaf; old approvals never transfer.

Config contains names/revisions and opaque independent host references only.
It carries no executable paths/arguments, environment names, credentials,
registration snapshots, private value hashes or storage state. One optional
session binding and at most one candidate-producing authentication save
permission are supported. Other fresh named contexts remain allowed. Current
policy, actual closure provenance, durable claims, credentials/session
revocation and browser interaction authority remain separate host obligations.

## Independent browser report and run evidence

`runevidence.ObserveBrowserReport` takes independently expected Config,
complete approved report inventory and trusted host witnesses, separately from
submitted unchanged report-v5 bytes. It checks every message claim/send and
request/ordinal/protocol/question identity, contained launch, original deadline,
complete current credential lease, joined transport/driver/Chromium/callbacks,
registration context closure and bounded session candidate/save/release order.
Typed errors, extraction failure, missing responses, cancellation and crashes
remain unknown after any possible dispatch; all following leaves stay unstarted.
A successful complete leaf remains known across a separate checkpoint failure.
Unknown cannot authorize retry, continuation or fresh-session fallback.

A positive non-dispatch witness must cover the complete same attempt/inventory,
all initial and continuation messages and positively joined ownership. It can
only establish eligibility to propose a distinct newly confirmed successor;
old approval/claims are never transferred. Missing reports and `not_started`
without that witness establish no successor eligibility.

`openudon.browser-run-evidence.v1` is a separate closed, value-free wire using the
frozen BrowserConfig declarations and unchanged report observation. Its verifier
requires exact expected Authority/Config/scope/dry-run mode/report bytes and
independent trusted witnesses. No self-reported trace, booleans or hash can
supply host authority; current state and actual containment provenance remain
Kinet checks. Existing RunEvidence and legacy BrowserConfig/wires stay unchanged.

The selected M51 executable profile is a complete flat **browser-only** inventory.
Mixed HTTP/fnct/browser packages remain review inputs and refuse execution plan/
Authority derivation before native admission or host access. Pure HTTP/fnct
profiles remain supported under their existing contracts. No multi-kind host
ABI is invented. Actual consumer isolation/interoperability remains M56/M52.

The exact frozen M51 corpus qualifies 34 metadata report/host cases independently.
Test adapters explicitly identify fixture-only worker/handoff/driver placeholders;
they establish no actual process, browser, credential or containment proof.
Credential revisions are opaque native symbolic revision IDs, not invented SHA
fields. An optional session with no reuse/save permission can describe a fresh
access binding; registration input binding remains optional report metadata,
while package construction still requires its exact native declaration.

Independent launch witnesses must carry the observed native containment's nonce,
worker identity/closure and driver closure. Expected Config and opaque fact
references cannot stand in for these actual facts. Reuse/save additionally need
an observed exact durable access lease and binding identity/generation/timestamps.
Observed permissions may narrow approved rights on missing/expired-to-fresh
fallback; that fallback stays before launch and all dispatch, on the same access
binding. A narrowed save permission cannot produce a candidate or durable save.
Fresh-only work may omit acquisition and requires no invented durable lease.
Actual M56/M52 adapters project native observed facts; the immutable test vectors
use explicitly labeled fixture-only projections and establish no containment.

The new evidence serializer validates closed Config, observation, timestamps,
identity and sandbox/dry/native posture before emitting bytes. In-flight denial
latches terminal uncertainty, so a subsequent success response cannot resurrect
the denied leaf. Existing completed leaves survive a separate checkpoint failure.

Every supplied access witness is checked even when access presence is optional.
It must match a completed acquisition before launch. Absent actual access uses
empty join and dispatch trace access identities; optional metadata alone does
not require a durable lease or acquisition.
It projects actual native owner/agent/profile/authentication digests, credential
revisions, origins and binding identity/generation/timestamps. Source identity
comes from the unique approved same-session authentication/action sources. An
auth-only plan with no derivable action-profile digest needs a separately
independent expected native binding projection anchored to Config's opaque
binding hash; unsupported or ambiguous identity refuses. The verifier never
decodes that hash, computes a new native binding algorithm or copies expected
facts as actual observation. Actual acquisition uses the exact leaf's name,
source and permissions; global union rights cannot authorize another leaf.
This exact acquiring-leaf name and source check also applies to a supplied
fresh access witness with no reuse or save permission. Fresh registration
metadata does not invent an authentication-bound native lease.

Actually reused saved access requires creation at or before report start, start
strictly before unchanged expiry, and each relevant named-session leaf start
within those same bounds. Later teardown/report finalization remains allowed.
Unused expired metadata during narrowed pre-launch fresh fallback is distinct
from actually reused state and creates no new timestamp authority or permission.

When execution stops, every dispatched leaf still in an unknown state becomes
terminal. A refused continuation or a stop concerning another leaf cannot
permit a late success to replace that uncertainty. Earlier completed leaves and
undispatched leaves retain their outcomes. Explicit human denial also withholds
subsequent session candidates and durable saving, including saving a candidate
staged by earlier successful authentication. Unrelated machine failures do not
cancel an otherwise valid one-use save permission.
