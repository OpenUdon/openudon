# Headless iCoT Authoring Engine

Prepare iCoT for a future local web interface by adding Phase A only: an
internal, driver-agnostic authoring engine that preserves current terminal
behavior. The engine must open empty, seeded, explicit-session, and resumable
state; expose JSON-marshalable frontier/readiness/source/action/preview
snapshots; apply one complete dependency-ready round; autosave provenance; and
write the same artifacts as terminal iCoT only after explicit human approval.

Reuse the existing elicitor graph, round, readiness, rendering, source,
browser, and verification contracts. Move artifact writing into one shared
transaction so terminal and engine paths retain identical source
revalidation, browser metadata, collision, rollback, and draft-cleanup
behavior. Prove byte parity on the runtime-only-render eval fixture and retain
browser verification revalidation.

Do not add `icot ui`, an HTTP server, React, folder browsing, published JSON
schemas, session CLI verbs, or live Browsertools author orchestration to this
phase.
