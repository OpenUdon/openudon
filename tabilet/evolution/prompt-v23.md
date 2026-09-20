# Local iCoT UI Server

Add Phase B over the A07 headless engine: `icot ui --example DIR` as a
single-workspace, IPv4-loopback-only web server with an experimental versioned
JSON transport and a small embedded read-only status shell.

Preserve explicit answers/from-example, resumable session, existing final
state, then empty-state startup precedence and the existing reviewed
API/browser source flags. Generate a 256-bit per-process capability token,
exchange its one-time bootstrap query for an HttpOnly SameSite=Strict cookie,
also accept bearer authentication, enforce the active Host and Origin, emit no
CORS permission, and set restrictive browser security headers. Port zero is
ephemeral, browser opening is default, opener failure is nonfatal, and
SIGINT/SIGTERM shutdown is bounded.

Expose health, cached snapshot, complete-frontier round, and explicit approval
routes. Compute exact SHA-256 revisions over snapshot/completion/write state,
serialize checks and engine mutations under one server lock, reject stale or
post-write mutations, and never accept caller-provided slots or evidence
sources. Retain A07 source refresh, incomplete approval, collision, rollback,
and artifact-writing behavior. Strictly decode one JSON document, reject
unknown fields, and cap bodies at 1 MiB.

Embed separate HTML, JavaScript, and CSS assets in the Go binary. The Phase B
shell shows paths, readiness/top issue, frontier/preview/completion state, and
formatted snapshot JSON, with no mutation controls. Keep React authoring,
folder browsing, multi-session/remote serving, TLS/accounts, persistent tokens,
LLM drafting, workflow execution, and live Browsertools orchestration for later
phases.
