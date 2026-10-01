# Supervising-product browser capture protocol

M93.1 implements the `openudon.browser-capture.v1` event/decision foundation;
M93.2 adds the internal authenticated/TOTP transport and M93.3 the registration
transport over the existing browser controllers. The public command, profile
import and complete browser qualification remain subsequent M93 work.

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
records. Native controller/capture fixtures and package transactions are later
M93 checks. Existing iCoT UI/control/terminal paths remain available during 5A.
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
bounds before launch and uses native registration protocol v4. The worker's
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
