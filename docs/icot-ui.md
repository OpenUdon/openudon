# Local iCoT UI Server

`icot ui` serves one explicitly named authoring workspace on `127.0.0.1`. It
wraps the same headless engine and transactional approval writer used by
terminal iCoT. Phase C adds an accessible embedded authoring and review shell
over the experimental API v2; it is not a remote service or a supported public
API.

```bash
go run ./cmd/icot ui --example ./examples/<name>
```

The command selects an ephemeral port by default, prints a one-time bootstrap
URL, and opens that URL with the platform browser. Use a fixed loopback port or
suppress browser opening when needed:

```bash
go run ./cmd/icot ui --example ./examples/<name> --port 8419 --no-open
```

The UI accepts the terminal authoring command's reviewed local source inputs:

```bash
go run ./cmd/icot ui --example ./examples/<name> \
  --from-example ./examples/eval/runtime-only-render \
  --api-source graphql:catalog=./schema.graphql \
  --openapi weather=./openapi/weather.yaml \
  --browser-profile status=./reviewed/status.browser.json \
  --browser-verification ./reviewed/status.live-check.json \
  --browser-registry ./reviewed/registry \
  --source-root ./provider-metadata \
  --network never
```

`--answers FILE` and `--from-example DIR` are mutually exclusive. Startup uses
the first available state in this order:

1. explicit `--answers` or `--from-example`;
2. `<example>/.icot/session.yaml`;
3. existing `<example>/project.md` or `workflows/intent.hcl`;
4. an empty authoring state.

The shell shows revision, last successful refresh, workspace paths, selected
sources, readiness, top issue, the complete current frontier, proposed file
actions and overwrite conflicts, project and intent previews, completion,
external-modification state, and cached snapshot JSON. Each frontier question
is an accessible required form control with its prompt, rationale, slot, and
optional recommendation. A recommendation can be copied into an empty answer,
but is never silently accepted.

One submit sends every answer in the current frontier with the exact displayed
revision. Incomplete client-side rounds focus the first missing control. Before
approval, the operator sees preview, action, readiness, and conflict state,
checks the review acknowledgement, and chooses either the separate final or
explicitly incomplete approval. An overwrite conflict also requires a separate
overwrite acknowledgement. No mutation is retried automatically.

The shell polls every two seconds while visible, pauses when hidden, refreshes
immediately when visible again, and backs off exponentially to 30 seconds after
errors. A stale response preserves unsent answers until the operator explicitly
adopts the new revision; displaced answers remain visible in a local archive.
Retryable failures reconcile with a snapshot and offer an explicit retry only
when the revision is unchanged. Domain rejection leaves controls editable;
indeterminate failure disables mutation; workspace drift requires restart; and
frozen final or incomplete completion remains inspectable. A polite live status
announces mutation progress and outcome. After success, focus moves to the
first question in the next frontier, the proposal-review heading when approval
becomes available, or the completion banner after approval.

## Local API v2

The experimental response version is `openudon.icot-ui-api.v2`. API v1 routes
are not served.

| Route | Method | Behavior |
|---|---|---|
| `/healthz` | `GET` | Unauthenticated liveness only. |
| `/api/v2/snapshot` | `GET` | Returns cached authoring state after checking workspace ownership. |
| `/api/v2/round` | `POST` | Applies one complete current frontier transactionally. |
| `/api/v2/approve` | `POST` | Performs the explicit final or incomplete atomic write. |

Canonical API paths accept bearer authentication. The browser shell uses
equivalent relative routes beneath its per-process path, where its scoped
cookie is accepted.

A successful response contains `version`, `revision`, `completed`, `workspace`,
`snapshot`, and, after approval, `write_result`. `workspace` includes
`externally_modified`; `snapshot.write_conflicts` contains the exact sorted
read-only preflight conflicts that require overwrite approval. Revision is a
`sha256:<hex>` digest over the snapshot, completion state, optional write
result, and workspace-modification state.

Snapshot responses include an `ETag`. Send the prior revision through
`If-None-Match`; the server returns `304` when cached state and workspace status
are unchanged.

Every mutation must send the exact current revision:

```json
{
  "revision": "sha256:...",
  "answers": [
    {"question_id": "boundary.outcome", "value": "Return the reviewed report"}
  ]
}
```

Round callers cannot provide slots or evidence sources. OpenUdon binds the
current frontier's authoritative slots and records the answer as human input.
Approval is separately explicit:

```json
{
  "revision": "sha256:...",
  "human_approved": true,
  "allow_overwrite": false,
  "approve_incomplete": false
}
```

Successful final and incomplete writes freeze that process. Later mutations
return `409`, while snapshot inspection remains available. The engine also
fingerprints its project, draft, final intent, review metadata, and selected
materialized source targets. If an editor or second process changes one, the
revision changes, `workspace.externally_modified` becomes true, and every later
mutation returns `409 workspace_changed`. `allow_overwrite` never bypasses this
revision-bound drift check. Restart `icot ui` after reviewing the external
change.

Mutation bodies are limited to 1 MiB and accept only `application/json` with no
charset or UTF-8. Invalid UTF-8, duplicate object names at any depth, unknown
fields, malformed JSON, and multiple documents return `400`; oversized bodies
return `413`; unsupported media types or charsets return `415`.

Errors contain a closed code, safe message, `retryable`, and an opaque
`request_id`. Authenticated state errors also include the current `revision`.
State conflicts are `409`, domain rejection is `422`, and operational or
indeterminate failures are `500`. Only 500-class failures are logged, using the
request ID, route, stage, and a sanitized cause.

## Transaction And Workspace Guarantees

A round builds its complete prospective state and snapshot before atomically
saving the resumable draft. A pre-persistence error changes neither engine
state nor draft bytes. Once persistence starts, request cancellation does not
interrupt finalization.

Approval similarly builds the exact refreshed snapshot and prepared write plan
before commit. `write_result` is constructed afterward directly from the commit
outcome, with no fallible post-commit refresh. The shared writer validates the
complete plan before creating directories, temporary files, or backups; it
rejects duplicate, case-insensitive-equivalent, ancestor/descendant, and
remove/write path collisions. Selected sources cannot target `project.md`,
either intent path, or `.icot/**`. The writer then rechecks the accepted workspace fingerprint
immediately before replacement, rejects descendant symlinks, keeps every output
beneath the canonical example root, cleans temporary backups after successful
or successfully rolled-back transactions, and reports rollback failure as
indeterminate. If post-commit backup cleanup itself fails, approval still
returns the known successful frozen result and reports the housekeeping problem
through `write_result.cleanup_warnings`. Empty-directory pruning is best effort
and cannot turn a successful artifact commit into a failed request.

Workspace fingerprinting is optimistic rather than a persistent lease. Cached
inspection remains available after ordinary mutation rejection and after
external drift. An unreadable or unsafe watched path fails closed as an
operational error and may be retried after the filesystem problem is repaired.
Pre-refresh observation is bounded to the current watched paths and targets in
the current local/registry materialization plans; unrelated workspace files are
not read. SHA-256 is streamed with cancellation checks and file identity,
type, size, and modification state are verified around hashing. A path first
produced by refresh that was not observed is conservatively treated as
missing, so an existing or concurrently created target becomes drift instead
of a newly adopted baseline.

## Security Boundary

Every process generates a fresh 256-bit internal capability token, a separate
unguessable `/.icot-ui/<instance>/` browser path, and a random 12-character
Crockford Base32 access code. The browser opens a tokenless loopback root; the
code is shown only in the terminal. A same-origin POST to the exact instance
root exchanges it for an HttpOnly `SameSite=Strict` cookie and redirects to the
clean instance path. The code expires after five minutes, is single-use, and
allows at most five failed attempts per minute. API v2 request and response
shapes are unchanged. Non-browser local clients may still use the internal
bearer token on canonical API routes; it never appears in browser-open argv.

The server always binds IPv4 loopback and requires the active listener's exact
Host plus its exact Origin when an Origin is supplied. It emits no CORS
permission. Responses are no-store and carry restrictive CSP, COOP, CORP,
framing, referrer, content-type, and permissions headers. Request headers are
limited to 32 KiB; the server uses a five-second header timeout, 15-second read
timeout, and 30-second idle timeout. It intentionally has no global write
timeout because reviewed-source refresh can be long-running.

The UI does not execute workflows or start Browsertools live authoring. Stop
the server with `Ctrl-C`; SIGINT and SIGTERM trigger bounded graceful shutdown.

## Phase C Limits

Phase C has no folder browser, React or Node dependency, multi-session hosting,
remote/LAN serving, account identity, UI-owned LLM drafting, workflow
execution, or live browser orchestration. The shell remains plain embedded
HTML, CSS, and JavaScript. One process owns one example for one trusted local
operator.

## Browser Qualification

The required provider-free browser gate is:

```bash
make icot-ui-browser-check
```

It launches sandboxed Chromium through a build-tagged, test-only Playwright-Go
harness and exercises the actual loopback listener. It is also part of
`make release-saas-check`. A local host whose AppArmor or user-namespace policy
prevents Chromium sandbox startup may use
`OPENUDON_ICOT_UI_BROWSER_DISABLE_SANDBOX=1` for this test only; release
automation never disables the sandbox. That local override proves the browser
journeys but does not satisfy A11's pending hosted sandboxed release-runner
verification.
