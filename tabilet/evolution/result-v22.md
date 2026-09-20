# Headless iCoT Authoring Engine Result

OpenUdon A07 adds `internal/icot/engine` as a reader/writer-free internal
driver boundary. It opens local authoring state with bounded API/browser source
discovery, returns JSON-marshalable snapshots, applies only complete current
frontier rounds, autosaves resumable state, renders previews without final
writes, and requires explicit human approval before committing artifacts.

Terminal iCoT and the engine now share `internal/icot/artifactwriter` for
source and browser-verification revalidation, materialization, browser
capability/authentication metadata, collisions, draft promotion/cleanup, and
rollback-capable atomic writes. Runtime-only-render engine output matches the
existing CLI's `project.md` and `workflows/intent.hcl` byte for byte. Browser
tests retain value-free review summaries and reject a verification report that
changes between preview and approval.

Post-review hardening makes every approval refresh transactional and retry-safe,
requires registry-backed selections to be freshly rediscovered with matching
coordinates and digests, derives answer routing from the current frontier,
returns deep-cloned snapshots, and exposes every prepared write/removal in the
approval action list.

The resume path now preserves draft operation-detail refs and structured draft
events through `session.interview.evidence` alongside assumptions,
annotations, mappings, and decision evidence. Draft YAML serialization also
retains the shared interview contract's JSON wire names, fixing round-answer
node IDs on reload.

Phase A remains internal and CLI-compatible. It adds no web server, frontend,
folder browser, published schema, new session verbs, or live browser-authoring
authority.
