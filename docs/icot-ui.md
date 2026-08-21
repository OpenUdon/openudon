# Local iCoT UI Server

`icot ui` is OpenUdon's primary interactive authoring and review surface for
API and existing-account browser workflows. It serves one explicitly named
workspace on `127.0.0.1`, uses the same transactional engine as terminal iCoT,
and exposes the experimental `openudon.icot-ui-api.v3` wire. It is neither a
remote service nor a supported public API.

```bash
install -d -m 0700 /private/operator/openudon-authoring
go run ./cmd/icot ui \
  --example ./examples/<name> \
  --private-root /private/operator/openudon-authoring
```

`--private-root` is needed only for an API upload or browser capture. It must
be absolute, mode `0700`, non-symlink, and outside the example. Use
`--driver-dir` when the installed Playwright-Go driver is outside its normal
location. Chromium is an installed prerequisite; iCoT never downloads it.
A fixed `--port` and `--no-open` remain available for local automation or an
SSH tunnel to the loopback page.

## Journey and acquisition

The first human decision selects one starter and records its goal as decision
evidence:

- `api`
- `existing_account_sign_in`
- `authenticated_action`
- `existing_reviewed_capability`
- `freeform_mixed`

Older resumable sessions without this field remain valid. Registration,
account creation, consent, enrollment, CAPTCHA, recovery, billing, and
password-change discovery are unsupported; the shell shows guidance instead
of launching those flows.

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
point. The server copies its own executable under the private root, revalidates
that copy, and launches it as a separate process group using
`browsertools.author-session.v2`. Browsertools owns the Playwright-Go Chromium
context. Neither the iCoT engine nor the HTTP server initializes Playwright
in-process.

Before launch, a 30-second isolated doctor reports the installed Playwright and
Chromium readiness. Snapshot polling continues while the doctor runs. One
capture may be active at a time. During an active capture the UI remains
inspectable, but journey, source, interview, approval, and package mutations
are blocked. The UI stores and serves only Browsertools' UI-safe doctor shape;
the executable path and other private local paths remain CLI-only diagnostics.

The API and shell model these states explicitly: `preflight`, `configuring`,
`launching`, `authentication`, `human_input`, `exploration`,
`action_approval`, `completion_review`, `stage_review`, `staging`, `staged`,
`canceled`, and `failed`. The shell renders only reduced, validated protocol
data:

- Browsertools-issued candidate actions as buttons;
- exact origin, action, candidate, and POST-budget approval cards;
- credential checkpoints that instruct the operator to type only in Chromium;
- only the MFA kinds reported by Browsertools;
- typed completion and bounded output decisions.

Passwords, OTPs, cookies, storage, request bodies, raw Browsertools output,
child stderr, signing keys, runtime credentials, and the private result path
have no HTTP or DOM representation. The child environment is allowlisted.
Process cancellation terminates descendants and drains stdout. Browser-active
operations retain Browsertools' ten-minute bound; unanswered human checkpoints
cancel after 30 minutes and every capture has a two-hour absolute ceiling.

A successful capture is not staged automatically. OpenUdon stable-reads and
independently reconstructs the private result, then a separate Stage action
atomically creates one collision-free authentication/capability profile pair
and updates the safe
`openudon.authenticated-browser-authoring-review.v3` collection. A valid v2
singleton is migrated on that write. Existing profile files are never
overwritten. Capture staging does not select a workflow action or create final
`.icot/browser-sources.json`/`.icot/browser-authentication.json`; final
authoring approval owns that evidence.

Failed, canceled, expired, crashed, or malformed captures write nothing to the
workspace.

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

API v2 routes are removed. API v3 provides authenticated routes for snapshot,
journey selection, interview rounds/reopening, source upload/stage/removal,
browser preflight, capture start/respond/stage/cancel, authoring
approval/resume, package build, and closed-allowlist artifact inspection.
There is deliberately no registration, approval-generation, credential, run,
or execution route.

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
