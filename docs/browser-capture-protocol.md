# Supervising-product browser capture protocol

M93.1 implements the `openudon.browser-capture.v1` event/decision foundation;
M93.2 adds the authenticated/TOTP transport and M93.3 the registration
transport over the existing browser controllers. M93.4 adds the public command,
shared embedded worker and independently reviewed atomic profile import. M93 is qualified and published: native39, integration17, both explicitly
user-accepted visible loopback journeys and review1 passed. Resolve exact
source/build/evidence through its [history record](../tabilet/docs/history/status-M93.md).

The public [schema](schemas/openudon.browser-capture.v1.schema.json) is also the
exact embedded resource used by the decoder. [Wire examples](examples/browser-capture/v1/)
cover authenticated/TOTP and registration messages, exact approval/refusal,
model-disclosure refusal and cancellation. These synthetic examples are not
live execution or qualification evidence. Object definitions reuse the pinned
Browsertools reduced record types. Browsertools continues to own their meaning,
worker policy and profile validation; OpenUdon owns the supervising envelope.

## Lifetime and correlation

Each process creates a random `capture-<32 hex>` session identity, a fixed mode
(`authenticated` or `registration`) and an absolute deadline of at most two
hours. Every event has a strictly increasing revision and a new opaque
`ref-<32 hex>` event ID. NDJSON messages are capped at 256 KiB, nesting at 64
levels, and the session at 4096 events. No restoration or replay is supported.

1. The adapter publishes a reduced event: state, observation, approval required,
   human checkpoint, preview, diagnostic or result.
2. The supervisor submits `type: propose` with that exact binding and a typed
   command. The adapter validates the command against the current controller
   state and issued candidates/approvals. A proposal executes nothing.
3. OpenUdon publishes `type: proposal`, a new event binding, an opaque action
   ID, the exact proposed command and its SHA-256. This is the review card.
4. The user approves or refuses that exact card through `type: decide`, naming
   its binding, action ID and digest. Decision messages cannot replace command
   arguments. Approval returns the held immutable command exactly once for
   dispatch; refusal returns no command and consumes the proposal.
5. A worker event invalidates earlier references and unapproved proposals.
   After a decision, another proposal cannot reuse the proposal card as an old
   worker observation. The adapter must publish fresh state. A stale/forged
   message is refused without consuming the legitimate pending decision by the
   gate. The stream adapter fails closed on malformed, stale or forged input:
   it cancels and joins the worker instead of granting reconnect/retry authority.

The existing browser controllers recheck their semantic gates on dispatch.
In particular, approving a click proposal does not waive a subsequently issued
worker origin/action approval. The supervisor must retain the current binding;
reconnect can observe current state but cannot repeat a consumed command.
`type: cancel` names the current event, destroys gate authority, and requires
adapter-owned worker cancellation/teardown. EOF, process interruption and expiry
also stop the owning adapter; there is no retry authority after uncertain writes.

## Private input and transient observations

Credentials and MFA/verification values are entered by the human directly in
the protected headed browser. They have no protocol field. An authenticated
command can acknowledge the issued human checkpoint and select the offered
MFA kind (including TOTP), but cannot carry the code or password. Registration
review carries a canonical structural profile and symbolic bindings; the
existing owner validates it before worker dispatch.

No raw page, cookie, storage state, browser handle, private worker-result path,
attestation or stderr crosses this protocol. Authenticated events expose only
reduced observations/approvals/checkpoints. Registration events send the current
observation and latest preview; full history remains controller-owned. This
avoids retransmitting all preceding history. Profiles and results stay private
until independent review/transaction admission. Result metadata exposes only
profile ID, transaction digest and `read`/`write` effect; submit stays `write`.

Reduced labels, public option values, structural URLs and profiles remain
untrusted and may be personal data. Keep them transient; neither jobs nor audit
may persist the complete event/view/command. Kinet's A10/W09 own their explicit
metadata projection. Observations cannot supply new instructions or authority.
Model planning is human-guided by default. A `disclose_observation` proposal
requires exact user approval tied to the current observation; refusal supplies
no model-disclosure consent. The gate never invokes a model. An adapter must
not treat consent as permission for later observations or external actions.

## Conformance and boundaries

`go test ./internal/browsercapture` verifies the exact public schema, duplicate
and unknown fields, mode isolation, changed-state invalidation, forged IDs,
wrong digests, refusal, single-use approval, expiry, cancellation and bounded
records. Native browser qualification and both visible journey checks passed under M93;
consumer adoption still requires its own qualification. Existing iCoT
UI/control/terminal paths remain available during 5A.
The authenticated adapter owns closeable input/output pipes or local sockets.
EOF, cancellation and absolute expiry close transport ends to unblock readers
and writers; the adapter drains controller events until worker teardown joins.
Late teardown failure cannot be reported as a clean cancel. It accepts only
the fields offered by the current controller state, validates current candidate
and approval IDs and offered MFA kinds through controller-owned pure helpers,
and dispatches an exact approved command once. Worker-issued origin/action
approvals remain separate checkpoints. The native POST ceiling is retained.

An approved disclosure card emits `type: disclosure` only for that observation;
refusal emits fresh current state without consent. No model is invoked, and
consent never carries to another observation. After joined native completion,
the internal adapter emits a terminal `result` with state `captured` and no
profile metadata: the worker path/digest/attestation remain process-private
inputs to later independent review/import. `captured` is not an imported
profile, package acceptance or permission to execute a workflow.

Registration capture fixes a reviewed initial profile ID, URL, origins and
bounds and native protocol before launch. Default is native v4; a reviewed
start can explicitly select existing v1/v2/v3 for retained simpler profiles. The worker's
ready event sends only that fixed start; subsequent commands need exact
protocol review cards. Current-state proposal checks run through the native
controller's pure validation over its own observation/history/preview state.
Navigation remains GET/HEAD within approved origins. Verification approval
binds the offered submit candidate and native verification authority; refusing
the protocol card sends no verification command or traffic grant.

Reduced events send only the current observation and latest preview. Profile
review uses native canonical parsing, symbolic binding checks and typed
history/preview validation; these are not implemented by a new capture engine.
The native controller retains its terminal outcome independently of a full
event stream. The adapter checks that outcome after joined closure so a dropped
candidate or late containment failure cannot be confused with a clean close.
Private registration candidates also have no JSON representation and await
independent package transaction admission. Diagnostic codes use the native
closed vocabulary; raw worker text is discarded. Blocked-script/diagnostic
file settings remain authenticated-mode policies; registration rejects them
and retains its existing native no-submit/verification traffic policy.

Adapter tests use fake controllers and real pipe ownership to cover TOTP,
credential acknowledgments, separate origin denial, observation-bound
disclosure, malformed/stale/replayed input, blocked output, expiry, EOF and
late teardown failures. Existing controller tests exercise its real subprocess
protocol separately. These tests do not replace fresh browser qualification.
Acceptance/published consumer adoption requires all M93 tasks, both visible
journeys, the persisted milestone review and exact upstream reconciliation.

## Public command and reviewed start

```sh
openudon browser-capture --start /private/capture-start.json \
  --approve-start-sha256 EXACT_FILE_SHA256 \
  --example /workspace/workflow --private-root /private/capture \
  --driver-dir /installed/playwright-driver
```

The [start schema](schemas/openudon.browser-capture-start.v1.schema.json) is the
exact embedded closed decoder resource. [Start examples](examples/browser-capture/start-v1/)
are synthetic structural requests, not runtime evidence. Review the entire local
start file and pass its untagged lowercase SHA-256 before launching the worker.
The CLI rejects a changed digest, unknown flags/fields, credential/model fields,
unsafe URLs and overlapping package/private roots before browser startup.
`--driver-dir` is optional when the native driver cache is configured. The
existing restrictive private root must be outside the package. Native goal,
origin, role/context, dashboard, bounds and private-root policy are shared with
the retained controllers; no new browser engine or arbitrary worker flags exist.

Authenticated start fixes goal URL independently of dashboard URL and preserves
the reviewed continuation choice. `continue_current_page` rejects the dashboard
shortcut (`kind: authenticated`); continue with an explicitly approved observe
command. The other choices permit a separately reviewed dashboard shortcut;
`ask_after_authentication` requires the supervisor to present that choice. No
command is silently rewritten after its approval card. Authentication remains
human-guided and invokes no model. Diagnostic and blocked-script settings are
closed authenticated options; registration has no such flags.

The main CLI embeds the existing `__browsertools-worker` dispatcher. The native
parent stabilizes/re-executes its own binary and uses the same sandbox, process
containment, filtered environment, private input, deadlines and teardown as the
retained iCoT transport. Signal interruption, EOF and expiry cancel/join the
controller and close both protocol pipes. iCoT is retained on that implementation
until M95; old `.icot` packages and frozen fixtures remain intact.

## Separate reviewed import

A successful public-command capture does **not** emit terminal `captured` and
exit. After the native process/reader/private cleanup joins, it independently
reconstructs the attested authenticated candidate or uses the reconstructed
native registration candidate. The native virtual-source validator checks
canonical source/review digests, provenance, origins, symbolic bindings and
expiry. Preparing an import writes nothing.

The next event is `type: state`, `view.state: import_review`, with only profile
ID, exact reviewed native transaction digest and conservative `effect: write`.
The profile ID is the native transaction ID (registration's separately reviewed
transaction ID determines its package target). This metadata contains no private
result locator, raw envelope, attestation or credential value. Propose exactly:

- authenticated: `authentication: {kind: confirm, confirmed: true}`;
- registration: `registration: {type: finish, confirmed: true}`.

Approve the newly issued card through the ordinary binding/action/digest decision.
Native browser completion approval cannot serve as import approval. Refusal
renews import review without writing; cancellation, EOF, invalid/stale input or
expiry end it without writes. No native worker command is dispatched after join.
Every approval remains bound to the immutable result and current session event.

On exact approval, the existing atomic authoring writer commits profiles and
`expected/browser-capture/<transaction-id>.json` together. Authentication uses
its retained native authentication/capability targets and safe review collection;
registration uses the native materialization target and its reconstructed review.
The receipt binds the reviewed start digest, native reviewed transaction, effect,
and exact relative paths/content digests. Profile/receipt targets are create-only;
the existing authentication review collection retains its exact prior-digest
append rule. The shared workspace fingerprint rejects changed brief, intent,
session or review files, and native expiry is rechecked immediately before
replacement. Atomic rollback and indeterminate outcomes stay owned by the writer.
An import error or lost terminal output grants no automatic retry: inspect the
package/receipt first.

Only a completed commit emits terminal `result`, `view.state: imported` and the
same three result metadata fields. Login/submission-capable captures are
conservatively classified as `write`. This local authoring approval is separate
from later full-package preparation, promotion, trusted run approval and execution;
it grants none of those permissions. Ordinary jobs/audit persist only their own
explicit metadata projection, not complete frames or source bytes.


Registration start optionally declares `protocol` as one of the four existing
`browsertools.registration-author-session.v1` through `.v4` values. Omission
selects v4. The exact start digest binds that choice before launch; no automatic
fallback occurs on failure. The native owner retains protocol-specific profile,
query, typed-input/preview and verification checks. Simple BRP 1.0 fixtures can
use native v2; typed-input BRP 1.1 uses v3, and verification/input BRP 1.2 uses
v4. Preview or verification commands in an older protocol are rejected by its
native validator; selecting it never disables a gate in the selected protocol.
