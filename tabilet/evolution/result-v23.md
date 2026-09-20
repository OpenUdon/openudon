# Local iCoT UI Server Result

OpenUdon A08 adds `icot ui` as a standard-library-only local server over one
A07 engine and one explicitly selected example. It preserves deterministic
seed precedence and reviewed source inputs, binds only `127.0.0.1`, supports
ephemeral or fixed ports, opens the platform browser by default, treats opener
failure as a warning, and shuts down gracefully on SIGINT or SIGTERM.

The experimental `openudon.icot-ui-api.v1` transport exposes unauthenticated
liveness plus authenticated cached snapshot, complete-frontier round, and
explicit approval routes. A per-process 256-bit token is exchanged from the
bootstrap query into an HttpOnly SameSite=Strict cookie scoped beneath an
unguessable per-process path, preventing ordinary sibling loopback services
from receiving it and separate iCoT UI processes from overwriting one another;
local API clients may use bearer auth. Exact listener Host and Origin checks,
no CORS permission, strict 1 MiB one-document JSON, unknown-field rejection,
and restrictive security headers keep the boundary local and closed.

One server lock serializes revision checks and engine mutations. Revisions are
SHA-256 digests of cached snapshot, completion state, and optional write
result. Stale requests never call the engine, concurrent same-revision
mutations admit one winner, and any approved final or incomplete write freezes
future mutation while preserving inspection. Round inputs contain only
question IDs and values; the A07 engine remains authoritative for frontier
slots, human evidence, source refresh, preview, and atomic artifact writes.
Detached bounded synchronization after mutation errors prevents request
cancellation from leaving an advertised stale revision; a failed refresh
makes the session fail closed. Domain errors remain 422 while operational
context and filesystem failures return 500.

Separate embedded HTML, JavaScript, and CSS assets provide a read-only shell
for workspace paths, readiness/top issue, frontier count, preview/completion,
and formatted snapshot JSON. HTTP/direct-engine parity retains byte-identical
`runtime-only-render` project and intent files, and real browser verification
replacement and registry disappearance failures cross HTTP without writes or
revision corruption.

Phase B adds no remote/LAN service, TLS or accounts, persistent token, folder
browser, React/Node dependency, multi-session hosting, UI-owned LLM drafting,
workflow execution, or live browser-authoring authority.
