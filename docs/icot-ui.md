# Local iCoT UI Server

`icot ui` serves one explicitly named authoring workspace on
`127.0.0.1`. It wraps the same headless engine and approval writer used by
terminal iCoT. Phase B provides an experimental JSON transport and a small
embedded read-only status shell; it is not a remote service or a supported API.

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

The shell shows workspace paths, readiness, the top issue, frontier size,
preview availability, completion state, and formatted snapshot JSON. It has no
round or approval controls. A future authoring frontend may use the Phase B API,
but that wire has no compatibility guarantee.

## Local API

The experimental response version is `openudon.icot-ui-api.v1`.

| Route | Method | Behavior |
|---|---|---|
| `/healthz` | `GET` | Unauthenticated liveness only. |
| `/api/v1/snapshot` | `GET` | Returns the cached authoring snapshot and revision. |
| `/api/v1/round` | `POST` | Applies one complete current frontier. |
| `/api/v1/approve` | `POST` | Performs the explicit final or incomplete atomic write. |

Those canonical paths accept bearer authentication. The browser shell uses
equivalent relative routes beneath its per-process bootstrap path so its
path-scoped cookie is never accepted on the host-wide canonical routes.

A successful API response contains `version`, `revision`, `completed`,
`workspace`, `snapshot`, and, after approval, `write_result`. Revision is a
`sha256:<hex>` digest over the cached snapshot, completion state, and optional
write result. Every mutation must send the exact current revision:

```json
{
  "revision": "sha256:...",
  "answers": [
    {"question_id": "boundary.outcome", "value": "Return the reviewed report"}
  ]
}
```

Round callers cannot provide answer slots or evidence sources. OpenUdon binds
the current frontier's authoritative slots and records the answer as human
input. Approval is separately explicit:

```json
{
  "revision": "sha256:...",
  "human_approved": true,
  "allow_overwrite": false,
  "approve_incomplete": false
}
```

Successful final and incomplete writes freeze that process. Later mutations
return `409`, while snapshot inspection remains available. Stale requests also
return `409` without calling the authoring engine. JSON mutation bodies are
limited to 1 MiB, require `application/json`, reject unknown fields, and must
contain exactly one document. Error envelopes use the same API version with a
closed code and safe message; malformed input is `400`, authentication failure
is `401`, rejected Host or Origin is `403`, unsupported methods are `405`,
engine/domain rejection such as validation or a target collision is `422`,
and operational failure such as cancellation or filesystem I/O is `500`.
After any round error, the server refreshes its cache with a detached bounded
context. If that refresh fails, inspection and mutation fail closed instead of
advertising a stale revision.

## Security Boundary

Every process generates a fresh 256-bit capability token and derives a separate
unguessable `/.icot-ui/<instance>/` browser path. The printed startup URL
carries the token once as `?token=...`; that instance path validates it, sets
an HttpOnly `SameSite=Strict` cookie scoped to the same path, and redirects to
the clean instance path. The cookie is therefore neither sent to an ordinary
sibling loopback service nor overwritten by another iCoT UI process on a
different instance path. Non-browser API clients may instead send
`Authorization: Bearer <token>` to the canonical routes. The token is required
for the shell, assets, and every API route except `/healthz`.

The server always binds the IPv4 loopback address. There is no LAN bind option,
TLS termination, account system, persistent token, CORS permission, or
multi-user session. Requests must use the active listener's exact Host, and any
Origin header must equal its exact `http://127.0.0.1:<port>` origin. Responses
are no-store and carry restrictive content-security, framing, referrer,
content-type, and permissions policies.

The UI does not execute workflows or start Browsertools live authoring. Source
revalidation, incomplete-draft policy, collision handling, and deliverable
writes remain inside the existing A07 engine and atomic writer. Stop the server
with `Ctrl-C`; SIGINT and SIGTERM trigger bounded graceful shutdown.

## Phase B Limits

Phase B intentionally has no folder browser, React or Node dependency,
authoring controls in the embedded shell, multi-session hosting, remote/LAN
serving, account identity, LLM drafting, workflow execution, or live browser
orchestration. One process owns one example for one trusted local operator.
