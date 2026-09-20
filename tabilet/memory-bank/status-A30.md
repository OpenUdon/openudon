# A30 — Verification authoring and integration

| Item | State | Notes |
| --- | --- | --- |
| A30.1 | `[+]` | Authoring-authority v2, explicit provider/dependency review and author-session/result v4 in iCoT. |
| A30.2 | `[+]` | Versioned transaction/package binding and BRP/call 1.2 promotion integration. |
| A30.3 | `[+]` | Owner tests, focused smoke and bounded review closure. |
| A30.4 | `[+]` | Source repair, owner tests and focused local browser validation pass. Published exact dependencies complete 39 fresh native stages, three W8M journeys, independent verification and exact tested-byte adoption under W16.4i.4. Integration review iteration 3 closes with no open P1/P2; prior failures and consumed invocations remain preserved. |

The approved September 13 verification release covers Turnstile, reCAPTCHA v2
and hCaptcha, including invisible widgets activated by an approved Submit.
Background readiness advances to final approval; actual challenges remain human.
Keep production protection enabled. Detection proposes metadata; explicit review
grants bounded provider traffic separately from exact application destinations
and the one approved application POST. Tokens stay inside the browser.

Dependency order: UWS BRP/call 1.2 (tracked by W8M W16.4f) -> Browsertools A13
and Browserdriver M13 -> Udon M41 -> OpenUdon A30/E15 -> W8M W16.4g–k.
Preserve published 1.0/1.1 and UWS core. Add author-session/result v4, driver v6,
authoring-authority v2, operation-packet v3 and explicitly version expanded
review/transaction envelopes. Old consumers reject new versions before launch.

Provider policy binds HTTPS domain/path/method/frame rules and finite maxima:
256 requests, 32 MiB responses and 120 seconds, clamped by the operation deadline;
consumer authority may tighten them. Provider POSTs cannot consume or authorize
application submissions, popups, top-level navigation, downloads or other traffic.
Readiness is client evidence only; backend acceptance remains separate. Missing
or ambiguous widgets, expiry, unsupported integrations and early/duplicate POSTs
stop. Never replay a click, reload or automatically retry registration.

Each owner requires focused offline tests, one affected synthetic browser smoke,
privacy canaries and a bounded whole-diff review with no P1/P2 findings, maximum
ten iterations. Full acceptance v2 follows explicit dependency publication and
source freeze, once the integration candidate is ready; adopt exact tested bytes
and preserve the operating kit. Provider-network tests and real W8M authoring,
verification-only probing and registration require separate explicit authority.
The verification-only probe uses the trusted driver, no private inputs, zero
application mutations and clean teardown; it creates no registration claim.
Failure-report verification is distinct from successful-registration evidence.
reCAPTCHA v3/Enterprise assessments, solving services, fingerprint evasion and
production configuration changes are outside scope.

Implementation and bounded review iteration 2 pass with no open P1/P2 finding.
All normal Go tests, vet, sibling/API boundary and example checks pass with
published UWS/Browsertools module pins. Fresh synthetic iCoT review/package smoke
and the complete registration_ui_handoff smoke pass; the latter completed in
73.282 seconds against exact prepared Udon sources. No development cache hit
counts as qualification.

Integration review fixed protocol selection to use the reviewed call's exact
profile. Canonical source fields now select v6, while an unused packaged 1.2
profile cannot upgrade a legacy flow. The regression and affected owner suites
pass. E15's complete frozen qualification remains pending; no live provider or
W8M browser contact occurred.

E15 subsequently completed the fresh full acceptance-v2 gate and independent
verification; W8M W16.4h adopted the exact tested bytes. See
[status-E15.md](status-E15.md). Real provider and W8M operations remain gated.

## September 14 authoring-repair validation

The owner-source repair and focused validation pass under W8M W16.4i.3.
Browsertools/OpenUdon full module tests and vet, OpenUdon diagnostic/teardown
race tests, and 90 Browserdriver offline tests pass. An explicit private go.work
binds unpublished development sources; release module pins still need publication.

Fresh local synthetic checks cover all providers in both modes, native form
binding and rejected overrides, author/review/package promotion, rendered closed
failures and privacy canaries. The two failed authoring smokes, local property
isolation and corrected UI-test Host setup remain recorded with their outcomes.
Six form-property collisions are supported; masked getAttribute/hasAttribute DOM
APIs remain explicit no-submission failures in the pinned Playwright implementation.
No provider network, real account or fresh W8M attempt was invoked. All stage
supervisors verified teardown without force.

Evidence: `/home/peter/.local/state/w8m-browser/w16-authoring-repair-20260914-jltzggqq`.
See W8M status-W16.md, W16.4i.3 for source inventories, stage timings, preserved
failure history and validation details. Bounded source review iteration 2 has no
open P1/P2; existing evolution direction is retained. E15.5/W16.4i.4 still require
publication, frozen complete qualification and exact-byte adoption. No commit or
push occurred; the prior adopted kit and consumed English authoring remain intact.

## A30.4 qualified adoption closure — September 14

The authorized W16.4i.4 acceptance-v2 run passed in 4391.332 seconds
(73.2 minutes): 39 fresh native stages and three fresh W8M journeys,
with nine workflow receipts, three discarded local registrations and nine local
logins. Every journey reports zero unauthorized mutations, rejected retries,
fresh contexts, required session reuse and verified teardown. Development cache
results were not used as qualification.

The independent aggregate verifier and retained OpenUdon binary verify the
aggregate/native/offline evidence. Twenty frozen source inventories and all
eight runtime hashes across the three retained passes match. Pass one's exact
tested files are adopted without rebuilding. The disabled synthetic adapter
preflight passes without browser launch or application mutation. The required
sandbox helper's path, hash, root ownership and mode remain bound.

Private evidence: `/home/peter/.local/state/w8m-browser/w16-authoring-qualification-20260914-bgw53tdf`.
Acceptance SHA-256: `0457bd061aa367ae45b0fb767806501f64fd7f909dcb9500f4a0bf3733e69a17`.
Adoption SHA-256: `cdf956150bd6bb770277282c751f4336ccc9d03a837a951a5992361d5ca95be4`.
All 4,841 preservation hashes pass; earlier kits, failed local checks and every
consumed operation remain unchanged. Tofu's two inherited edits remain outside
these task commits. Later coordination records do not replace the frozen
execution-source pins.

This qualifies synthetic integration with application-request allowlists, not
network-wide containment or production provider acceptance. The
consumed English attempt's hidden live cause remains unproven. Masked
getAttribute/hasAttribute forms still stop without submission. No provider
fixture, W8M contact, real account operation or deployment occurred. Fresh
English authoring and the dependent verification-only probe still require new
exact authority. Existing evolution direction is retained.

Bounded integration review iteration 3 passes with no open P1/P2.
