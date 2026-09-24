# Retired milestone M30 - M30 - Build From Intent Hardening

**Milestone.** M30
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M30.md
**Source status SHA-256.** f17e863a3257d33a36664ecc718df83bb392879d11d50118535038dc480f56ce
**Source milestone snapshot.** tabilet/docs/history/milestone-before-legacy-retirement.md.txt
**Snapshot SHA-256.** 26884eeda9af4ded30e6d545a33dca84c401d2320c22355fe4fb0336d5dcac71
**Evidence.** 71a4f78afbcf2180fc478ffa89c53544c9160648
**Worktree.** includes uncommitted changes
**Review.** not established
**Review iterations.** not recorded
**Verification.** Original status bytes and full earlier milestone bytes preserved by SHA-256; no fresh acceptance claim.
**Consolidated into.** Current milestone dashboard and maintained memory-bank guidance; full earlier text remains in the frozen snapshot.

## Status record

~~~~~~~~~~~~~~~~~~~~markdown
# M30 - Build From Intent Hardening

## Goal

Make `openudon build` the dependable deterministic path from reviewed
`workflows/intent.hcl` to generated workflow, UWS, expected-plan, review,
handoff, refinement, and quality artifacts.

M30 is build-only. It does not change iCoT prompt behavior, `openudon
synthesize` intent generation, or trusted execution.

## Status

| Item | State | Notes |
|---|---|---|
| Milestone framed | `[+]` | M30 follows M29 and focuses on build from existing `intent.hcl`, not on authoring or execution. |
| Native operation index repair implemented | `[+]` | `build` quality/plan logic now indexes package-local Google Discovery and AWS Smithy operations, including dotted and normalized operation ID aliases. |
| Credential source policy repair implemented | `[+]` | Security-scheme quality accepts `credentials.<binding>` when `<binding>` is declared in project credential policy. |
| Opaque `$ref` response path handling implemented | `[+]` | Array item `$ref` response paths now warn as opaque metadata instead of failing as missing properties. |
| Discovery request-body parity implemented | `[+]` | Google Discovery request schemas now expose required body fields such as Gmail `raw` to iCoT review repair, build request mapping, expected-plan generation, and quality checks. |
| General local render-before-delivery repair implemented | `[+]` | Deterministic draft review can detect missing rendered request-body content for delivery/message sinks and insert a local `fnct` render step from the terminal upstream producer, without Toronto-specific logic. |
| Advisory security sidecars implemented | `[+]` | Build reads `*.security.{json,yaml}` and `*.security-overlay.{json,yaml}` files next to package-local API source files and uses them for credential/security quality and plan evidence without treating the sidecars as API source contracts. |
| Advisory sidecars packaged as evidence | `[+]` | Associated security sidecars are now included in handoff inputs, required package paths, and package digests while staying out of API-source operation discovery. |
| Weather/Gmail build sample checked | `[+]` | `go run ./cmd/openudon build --example ./examples/weather-toronto-gmail` passes locally after the native-source repair; `assess` passes with one opaque response-path warning. |
| Native required-parameter parity | `[+]` | Google Discovery and AWS Smithy body/parameter requirements now participate where local metadata exposes required path/query/header/payload/unbound body members, with optional Smithy members remaining non-blocking. |
| Native security parity | `[+]` | Intrinsic Google Discovery OAuth scopes and AWS Smithy SigV4 signing metadata now participate in build credential validation and plan evidence without adding runtime auth behavior. |
| Native response-field parity | `[+]` | Response path validation now consumes resolvable Google Discovery response refs and conservative Smithy output body schemas, while unresolved native response metadata remains warning-only. |
| Build offline dependency cleanup | `[+]` | `openudon build` now uses the deterministic build-from-intent package path without resolving an LLM/chat client; provider/model remain optional review-evidence labels only. |
| Focused build regression target | `[+]` | Added a focused `PackageFromIntent` build-only regression matrix covering OpenAPI, Google Discovery, AWS Smithy, multi-source, local no-source `fnct`, and invalid-operation negative packages. |

## Design Notes

- `intent.hcl` is the source of truth for M30. Build may normalize explicit
  intent metadata, but it should not make authoring decisions that belong to
  iCoT, `synthesize`, or the operator.
- Build quality should prefer precise failure for contradictory local metadata,
  precise warning for opaque local metadata, and no fabricated certainty.
- Typed API source support must stay inside OpenUdon's artifact/review layer.
  Runtime parsing and execution remain behind the trusted executor boundary.
- Delivery APIs that require rendered body content, such as message send
  operations, should receive that content through explicit local `fnct` steps.
  Build validates the explicit binding; iCoT review repair may propose the
  render step when local request-body metadata proves it is missing.
- Advisory security sidecars are package evidence, not execution credentials.
  They let build/review validate declared credential bindings for native source
  files whose source format does not carry OpenAPI-style security metadata, and
  they are digest-covered when present. iCoT may materialize matching sidecars
  during catalog migration; `openudon build` remains offline and only consumes
  sidecars already present in the package.
~~~~~~~~~~~~~~~~~~~~
