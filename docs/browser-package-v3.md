# Package v3 browser supplement

P10 adds an explicit browser supplement to the public `packagev3` byte API.
It consumes ordinary published Browsertools M33
`v0.0.0-20261010115646-1859f5e34367` and UWS C10
`v0.0.0-20261009220914-f01a2542410c`. The independent retained HCL codec stays
at `v0.0.0-20261008043726-b099f6803277`.

`SourceInput.Kind == "browser-profile"` admits exact native action,
authentication and registration profile bytes under `sources/browser-profile/`.
Build and Verify independently reproduce the complete shape table with
Browsertools; caller-created shapes or matching hashes alone establish no proof.
Selected digests cover complete lossless native action/flow subtrees, while
source digests separately cover every original byte, including shared contexts.
Native schema validity and runtime admission are separate checks.

Browser packages include `expected/browser.json` (`openudon.browser-supplement.v1`)
in the manifest/input digest and complete handoff inventory. The manifest's
optional `browser` reference and `browser_reviews` inventory bind that supplement
and its separately reviewed artifacts. Non-browser packages omit both fields
and preserve the existing package/assessment/handoff/plan bytes and digests.

The supplement binds the exact ordered browser leaves in one straight-line
workflow, source/selected hashes, per-leaf protocol pair, complete origins,
effects, confirmation-policy digest, symbolic credential names and declared
registration input binding. A browser call never carries credential values,
registration snapshots, browser state, local paths or driver arguments.
`BrowserBuildOptions.Permissions` makes explicit per-step reuse/save choices;
zero values select fresh contexts. Unknown step choices, registration reuse/save,
or action save permissions refuse. The host separately confirms current
credential/session authority and supplies isolated runtime admission.

`VerifiedPackage.BrowserSupplement()` returns an independent copy. The existing
execution plan adds an optional `browser` call only for browser leaves and
requires the host's implementing runtime admission before deriving their plan.
Old non-browser approvals never acquire browser authority by this addition.

These APIs perform no browser launch, external request or private-runtime import.
Kinet M56/M52 own actual containment/custody and later actual producer
interoperability qualification. Frozen synthetic M51 vectors are metadata
conformance inputs and grant no execution authority.
