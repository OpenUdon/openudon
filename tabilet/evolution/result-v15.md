# Result V15 - Value-Free Browser Verification Packaging

OpenUdon now accepts explicit `browsertools.live-check.v1` and
`browsertools.portability-check.v1` files through repeatable
`--browser-verification` inputs. Its downstream-only verifier strictly bounds
and decodes the public JSON, rebuilds the exact profile/action probe plan, and
checks digest, origin, lifecycle, types, counts, fixed diagnostics, Chromium
baseline, and cross-engine consistency without importing Browsertools capture
or Playwright code.

iCoT retains a report path only in resumable local state and reopens it at
approval. The final package contains normalized value-free facts and the source
digest in `.icot/browser-sources.json`, which is independently checked by
quality and human-review generation and already participates in canonical
inventory, digest, approval, and trusted handoff. Failed, private, stale,
mismatched, malformed, replaced, conflicting, or invented-success evidence
fails closed; no report is required and portability grants no runtime policy.

The milestone review fixed source-reference normalization, resume precedence,
malformed attachment elision, Browsertools generic-failure compatibility,
required/duplicate JSON handling, and unvalidated passed review rendering.
OpenUdon commit `81ede87` passed focused/race/full workspace, standalone,
boundary, document-memory, UWS validation, and Browsertools/UWS/Udon
compatibility gates. E01 now owns the provider-free end-to-end browser
integration evaluation.
