# Unified iCoT UI And Package Handoff Result

OpenUdon A15/A16 now implements the unified local authoring lifecycle. The
engine persists five journey starters, validates API-family uploads in a 20 MiB
private inbox, atomically stages/removes digest-owned sources, and stages
collision-free reviewed browser profile pairs with a deterministic safe-review
v3 collection and valid-v2 migration.

Experimental `openudon.icot-ui-api.v3` removes v2 routes and carries separate
authoring and capture revisions under one complete-state ETag. The embedded
shell handles acquisition, isolated Browsertools doctor/capture, ordinary
frontier authoring and review, authored state, separate package build/current-
byte assessment, failure remediation/resume/reapproval, and handoff-ready
allowlisted inspection. Sensitive browser/session/value fields and runtime
execution authority have no HTTP route or representation.

One `icot` executable now privately stabilizes and re-executes a hidden
Browsertools worker. UI and bundled terminal mode share a typed asynchronous
coordinator with minimal environment, process-group cancellation, stdout
draining, a 30-minute operator-idle bound, and a two-hour absolute ceiling.
Playwright remains Browsertools-owned and runs only in the worker process; the
engine and HTTP server never initialize it.

Focused tests cover API v2 retirement, separate revisions, non-blocking doctor
polling, capture locking/staging, private-root/source authority, review
migration/collisions, package failure recovery/reapproval, artifact allowlists,
and absent execution routes. E08 remains open for the complete installed,
sandboxed Chromium access-code-to-handoff journey and leak scan. Standalone
module verification also waits for the sibling Browsertools A06 commit to be
published and pinned; no local replace is introduced.
