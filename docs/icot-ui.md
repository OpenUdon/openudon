# Local iCoT UI Server

`icot ui` is OpenUdon's primary interactive authoring and review surface for
API and existing-account browser workflows. It serves one explicitly named
workspace on `127.0.0.1`, uses the same transactional engines as terminal iCoT,
and exposes the experimental `openudon.icot-ui-api.v4` wire. It is neither a
remote service nor a supported public API.

```bash
install -d -m 0700 /private/operator/openudon-authoring
go run ./cmd/icot ui \
  --example ./examples/<name> \
  --private-root /private/operator/openudon-authoring
```

`--private-root` is needed only for an API upload, browser capture, or guided
registration authoring. It must be absolute, mode `0700`, non-symlink, and
outside the example. Use `--driver-dir` when the installed Playwright-Go driver
is outside its normal location. Chromium is an installed prerequisite; iCoT
never downloads it. A fixed `--port` and `--no-open` remain available for local
automation or an SSH tunnel to the loopback page.

To review a public browser-profile transaction in the shell, supply the package
option group and the optional existing transaction:

```bash
go run ./cmd/icot ui \
  --example ./examples/<name> \
  --browser-transaction ./transaction.json \
  --package-scope examples/<name> \
  --package-scratch /absolute/restrictive-scratch-parent \
  --package-store /absolute/generation-store
```

`transaction.json` is a public value-free v1 or v2 artifact, not a private
Browsertools result or result path. The package paths stay process-private and
have no API/DOM representation. Omitting any member of the package option group
is a usage error. Omitting `--browser-transaction` starts the transaction
engine without a candidate so the guided registration-authoring path can adopt
one after clean Browsertools teardown.

## Guided browser registration authoring

The registration wizard is available only when `--private-root` and the full
package option group are configured. It uses authenticated, revision-bound API
v4 start, typed-command, and cancel operations to control one isolated headed
Chromium worker with Browsertools registration protocol v2. Authoring permits
GET and HEAD navigation only. It cannot type credentials, submit a form, create
an account, sign in, or execute the workflow it describes.

The operator reviews metadata, confidence and expiry, symbolic credential
slots, ordered declarative macro steps, observed accessibility locators,
effects, confirmation and success proof, fixed call controls, cleanup, and the
exact canonical retained structural query. Credential or verification values
have no request, API-state, or DOM field. The server constructs and revalidates
the canonical UWS registration profile; JavaScript does not supply profile
bytes.

Candidate adoption succeeds only after the worker closes and its process tree
is drained. The exact private candidate then enters the ordinary value-free
browser transaction review. Explicit review selects it as a virtual source;
the normal iCoT authoring rounds write the credential-free profile and review,
build and qualify the package, and only then permit transaction preparation and
atomic promotion. Canceling a pending candidate is terminal for that UI
process; start a fresh iCoT process for another registration-authoring session.

## Journey and acquisition

The first human decision selects one starter and records its goal as decision
evidence:

- `api`
- `existing_account_sign_in`
- `authenticated_action`
- `existing_reviewed_capability`
- `freeform_mixed`

Older resumable sessions without this field remain valid. General workflow
discovery for registration, account creation, consent, enrollment, CAPTCHA,
recovery, billing, and password changes remains unsupported; the shell shows
guidance instead of launching those flows. The separate bounded registration
wizard described above authors only an explicitly configured BRP and grants no
runtime authority.

API-family files enter a private, 20 MiB bounded inbox. iCoT rejects symlinks,
invalid UTF-8, secret-like content, unknown or ambiguous documents, and files
that do not produce exactly one validated Apitools source candidate. The shell
shows the detected family, operation count, digest, and canonical workspace
target. A separate Stage action performs the atomic write and refreshes engine
discovery. Removal is allowed only for a UI-staged path whose exact digest is
still owned by the UI; edited or externally replaced files fail closed.

Existing reviewed local sources and explicitly configured Browsertools static
registries remain available. There is no UI provider catalog, folder browser,
or desired-state conversion surface.

## Isolated authenticated browser capture

The distributed `icot` executable contains a hidden Browsertools worker entry
point. The server fully copies and hashes its executable into a private
content-addressed cache, publishes the digest-named entry create-only with mode
`0500`, revalidates every reused entry byte-for-byte, and launches it as a
separate process group using `browsertools.author-session.v2`. Only stale
regular temporary files with the owned prefix are swept. A hard interruption
during first publication can leave a bounded owned temporary, while completed
digest entries are reused instead of creating one stabilized executable per
capture.
Browsertools owns the Playwright-Go Chromium context. Neither the iCoT engine
nor the HTTP server initializes Playwright in-process.

Before launch, a 30-second isolated doctor reports the installed Playwright and
Chromium readiness. Snapshot polling continues while the doctor runs. One
capture may be active at a time. During an active capture the UI remains
inspectable, but journey, source, interview, approval, and package mutations
are blocked. The UI stores and serves only Browsertools' UI-safe doctor shape;
the executable path and other private local paths remain CLI-only diagnostics.

The API and shell model these states explicitly: `preflight`, `configuring`,
`launching`, `authentication`, `human_input`, `exploration`,
`action_approval`, `completion_review`, `stage_review`, `staging`, `staged`,
`canceling`, `canceled`, `closed`, and `failed`. A worker `closed` state is
terminal and does not trigger another observation request. The shell renders
only reduced, validated protocol data:

- Browsertools-issued candidate actions as buttons;
- exact origin, action, candidate, and POST-budget approval cards;
- credential checkpoints that instruct the operator to type only in Chromium;
- only the MFA kinds reported by Browsertools;
- typed completion and bounded output decisions.

The same typed parent controller serves UI, bundled terminal, expert external,
and loopback capture. Before an event enters HTTP state it validates the closed
message union, phases, canonical reduced labels, accessibility roles,
diagnostics, context/frame identifiers, challenge kinds, additive context
inventory, and negotiated bounds. Configured and page-derived paths pass the
shared disclosure validator first, so an unsafe value is rejected without
entering the snapshot, ETag, DOM, logs, or planner input.

A new canonical HTTPS or loopback HTTP origin is shown on the existing exact
approval card and is authoritative only after the response matches the pending
navigation or click. The accumulated ledger governs subsequent events. The
process-private parent attestation then binds the ordered actions,
observations, checkpoints, approvals, dashboard proof, output requests,
contexts, diagnostics, and final origin set. The attestation is neither
serialized nor exposed by API v4.

Passwords, OTPs, cookies, storage, request bodies, raw Browsertools output,
child stderr, signing keys, runtime credentials, and the private result path
have no HTTP or DOM representation. The child environment is allowlisted.
Process cancellation terminates descendants and drains stdout. Browser-active
operations retain Browsertools' ten-minute bound; unanswered human checkpoints
cancel after 30 minutes and every capture has a two-hour absolute ceiling.

A successful capture is not staged automatically. OpenUdon stable-reads and
independently reconstructs the private result, requires it to match the parent
attestation, then a separate Stage action atomically creates one collision-free
authentication/capability profile pair and updates the safe
`openudon.authenticated-browser-authoring-review.v3` collection. A valid v2
singleton is migrated on that write. Existing profile files are never
overwritten. Capture staging does not select a workflow action or create final
`.icot/browser-sources.json`/`.icot/browser-authentication.json`; final
authoring approval owns that evidence.

Failed, canceled, expired, crashed, or malformed captures write nothing to the
workspace.

## Browser-profile transaction review

When launched with a public transaction, the shell presents the exact BAP+BCP
or BRP composition, immutable origins/times/digests, symbolic bindings and
session posture, candidate output targets, registration cleanup, preparation
and qualification evidence, promotion/recovery identity, and selected-package
side-effect posture. BRP review includes Browsertools' canonical
accessibility-label heuristic/not-DLP disclosure and fixed GET/HEAD-only,
zero-mutation, no-submit, no-account, no-session, and no-runtime facts. Its
approval symbol is descriptive and grants no authority.

Review, scratch preparation, promotion, recovery acceptance, and cancellation
use separate checkboxes and revision/digest-bound requests. Stale revisions
force a refresh and fresh consent. Expired candidates cannot be reviewed or
prepared. Indeterminate promotion exposes the safe target and exact recovery
report; blind retry is unavailable. Promotion selects package bytes only. The
shell has no run, execute, sign-in, registration-submit, credential, or account
operation.

## Authoring, package, and handoff lifecycle

The UI uses one optimistic authoring revision for journey, source, interview,
write, resume, and package mutations, plus a separate capture revision for
asynchronous browser events. The response ETag covers both. Stale mutations
are never retried automatically. A changed workspace preserves inspection but
blocks mutation until restart.

The shell presents the complete frontier, closed controls, structured
deferrals, settled-decision reopening, source and capability provenance,
digests, origins, actions, expiry, the symbolic credential/browser-session
map, and a compact workflow graph. Final authoring approval atomically writes
`project.md`, intent, selected sources, and review metadata, then enters
`authored`; it does not finish the UI lifecycle.

“Build reviewed package” is a separate revision-bound confirmation. The local
deterministic build has a two-minute deadline. iCoT then performs a non-writing
assessment of the exact current bytes. A quality failure shows check-specific
remediation and permits an explicit return to authoring. That transition
clears the prior write/package result, and final authoring approval must be
repeated.

A passing package enters `handoff_ready` and freezes authoring. The shell shows
quality checks, side effects, symbolic credentials, runtime approvals, package
and handoff digests, the exact `openudon approval-template` argv, and bounded
artifacts from a closed allowlist. Snapshot polling and artifact reads recheck
the frozen artifact sizes and digests; drift invalidates handoff readiness and
requires resume, repeated approval, and rebuild. The UI does not create sandbox or production
approval, accept credential values, call `openudon run`, or execute a workflow.

## API and access boundary

The tokenless bootstrap accepts only the random 12-character terminal access
code. The code expires after five minutes, is single-use, and is rate-limited;
recovery prints a replacement only to the terminal. A successful exchange
creates a scoped HttpOnly `SameSite=Strict` cookie on an unguessable instance
path. Bearer authentication remains available for local automation. Exact
loopback Host and Origin checks, no CORS, restrictive headers, bounded strict
UTF-8 JSON/multipart decoding, and signal-driven shutdown remain mandatory.

API v2 and v3 routes are removed. API v4 provides authenticated routes for snapshot,
journey selection, interview rounds/reopening, source upload/stage/removal,
browser preflight, capture start/respond/stage/cancel, authoring
approval/resume, package build, and closed-allowlist artifact inspection.
It additionally exposes current/start/review/prepare/promote/cancel,
recovery inspection/reconciliation, and selected-package inspection for the
value-free browser transaction. There is deliberately no registration submit,
approval-generation, credential, run, or execution route.

## Qualification

The provider-free shell gate remains:

```bash
make icot-ui-browser-check
```

It uses a build-tagged test harness to exercise the actual loopback listener.
The unified acquisition-to-handoff qualification additionally requires an
installed Chromium run covering access code, starter, authenticated capture,
profile stage, resumed interview, repeated review as needed, package build,
and passing handoff. Real-site runs must use an operator-authorized
non-production tenant and retain only value-free evidence. The required target
rejects the sandbox-disable override, sets an explicit sandbox-required
control, and logs `chromium_sandbox_enabled=true` for every journey. Use
`make icot-ui-browser-check-unsandboxed` only to diagnose a local host whose
kernel policy blocks user namespaces; it cannot substitute for sandboxed
release evidence.
