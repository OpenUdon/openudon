# A18 Secret-Safe Browser Registration Intent And Packaging

Item | State | Notes
--- | --- | ---
A18.1 Offline registration package lifecycle | `[+]` | OpenUdon commit `07f5d482ae6fecbe462f5ccd5bc2eea7f10d0d3a` pins the published UWS and Browsertools contracts; explicit intent lowers to the registration call, strict digest/currentness/provenance review gates bind exact symbolic policy, package/review/handoff and side-effect evidence are complete, dry-run exports the reviewed approval ID without values, and non-dry executor construction fails closed. Workspace and standalone test/vet, standalone race, `make check`, UWS validation, doc-memory, strict MkDocs, diff review, and reachable-vulnerability gates pass without target access.

This milestone does not authorize or perform account registration, sign-in,
credential resolution, human verification, cleanup, browser launch, or target
access. A live registration run remains blocked on exact-account,
exact-origin-inventory, and exact-run approval plus compatible pinned Udon and
Browserdriver runtime support.
