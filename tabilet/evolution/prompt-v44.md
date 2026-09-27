# OpenUdon-owned non-interactive step authoring for Kinet

Approved planning direction, 2026-09-26.

Deliver sibling S2a from Kinet `docs/ideas.md` section 10 and `docs/icot.md`
sections 4–7: `openudon step candidates`, `step bind`, `step check`, and
`flow-review`, each with stable versioned JSON and published conformance
fixtures. Candidates expose ranked operations for a plain-language step
contract, consumer-readable summaries, authentication needs, and effect class.
Bind/check write or validate one step in `intent.hcl`; flow review exposes the
existing advisory behavior as a command.

OpenUdon must draft and own the contract now rather than wait for Kinet W03's
first row, which follows Kinet M05 and A03. Kinet consumes the published
contract and owns its workflow loop. Apitools retains operation metadata,
classification and ranking ownership. Keep S2b simulation/browser acquisition
and S3 iCoT retirement as candidates. Preserve current iCoT and package
compatibility while planning the additive surface.

The owner approved four planning-file actions: update the milestone dashboard,
specification, index and candidate entries; create `status-M87.md`; and create
this prompt/result v44 pair. Implementation and publication are separate.
