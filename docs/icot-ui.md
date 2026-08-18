# Local iCoT UI Server

`icot ui` serves one explicitly named authoring workspace on `127.0.0.1`. It
wraps the same headless engine and transactional approval writer used by
terminal iCoT. Phase B provides an experimental API v2 and a small embedded
read-only status shell; it is not a remote service or a supported public API.

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
source and proposed action counts, readiness, top issue, frontier size,
preview, completion, external-modification state, and cached snapshot JSON. It
polls every two seconds while visible, pauses when hidden, refreshes immediately
when visible again, and backs off exponentially to 30 seconds after errors. It
has no round or approval controls.

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
`externally_modified`. Revision is a `sha256:<hex>` digest over the snapshot,
completion state, optional write result, and workspace-modification state.

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

Approval similarly builds the exact refreshed snapshot and write result before
commit. The shared writer rechecks the accepted workspace fingerprint
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
Paths that first become engine-owned during a round are compared with a
pre-refresh workspace observation, so a concurrent editor cannot be adopted as
the new baseline.

## Security Boundary

Every process generates a fresh 256-bit capability token and derives a separate
unguessable `/.icot-ui/<instance>/` browser path. Only that exact instance root
may exchange the printed `?token=...` query for an HttpOnly
`SameSite=Strict` cookie, followed by a redirect to the clean instance path.
Non-browser clients may send `Authorization: Bearer <token>` to canonical API
routes. The token is required for the shell, assets, and every API route except
`/healthz`.

The server always binds IPv4 loopback and requires the active listener's exact
Host plus its exact Origin when an Origin is supplied. It emits no CORS
permission. Responses are no-store and carry restrictive CSP, COOP, CORP,
framing, referrer, content-type, and permissions headers. Request headers are
limited to 32 KiB; the server uses a five-second header timeout, 15-second read
timeout, and 30-second idle timeout. It intentionally has no global write
timeout because reviewed-source refresh can be long-running.

The UI does not execute workflows or start Browsertools live authoring. Stop
the server with `Ctrl-C`; SIGINT and SIGTERM trigger bounded graceful shutdown.

## Phase B Limits

Phase B has no folder browser, React or Node dependency, authoring controls in
the embedded shell, multi-session hosting, remote/LAN serving, account identity,
LLM drafting, workflow execution, or live browser orchestration. One process
owns one example for one trusted local operator.
