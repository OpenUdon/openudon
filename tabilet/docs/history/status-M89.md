# Retired milestone M89 - Stage 1 source provisioning and binding contract

**Milestone.** M89
**Outcome.** completed
**Retired.** 2026-09-28
**Source status.** tabilet/memory-bank/status-M89.md
**Source specification.** tabilet/memory-bank/milestone.md#m89--stage-1-source-provisioning-and-binding-contract
**Evidence.** 2e2ecedb32add53e10d6d81d4b2528ca2bdfdcc8
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 5
**Verification.** `GOWORK=off go test ./...`, `GOWORK=off make check`, the OpenUdon sibling check, APItools boundary check, `check-doc-memory`, UWS validation, and `git diff --check` passed. Kinet's focused `TestOpenUdonEmptyPackageJourneyWithNestedMappings` and exact clean checkout-backed `GOWORK=off KINET_OPENUDON_DIR=../openudon make openudon-check` passed against clean published OpenUdon revision `2e2ecedb32add53e10d6d81d4b2528ca2bdfdcc8`.
**Consolidated into.** [milestone index](../../memory-bank/milestone.md), [architecture](../../memory-bank/architecture.md), and [technical stack](../../memory-bank/tech-stack.md). No product, reusable-lesson, or evolution change was required.

## Milestone specification

````markdown
## M89 — Stage 1 source provisioning and binding contract

Using accepted and published APItools M79 revision
`e3625f6ef52ea54b7f78b7a4a4f1993bf8a06a46`
(`v0.0.0-20260928033144-e3625f6ef52e`), let Kinet provision explicitly
confirmed local API documents through an OpenUdon-owned package-write
command. Validate bounded local inputs through APItools-owned parsing and
materialization, preserve source digests/provenance, and do not fetch implicit
remote sources. Support explicit nested/renamed request and response mappings
with source/type/requiredness evidence. Continue to refuse `unknown` effects
and unsupported root extensions with diagnostics naming the precise
unsupported input. Acceptance includes standalone tests, the exact APItools
revision binding, package-root-safe CLI use, and a consumer handoff to Kinet
W04. This Stage 1 remediation is distinct from the S2b Stage 5 candidate.
````

## Status record

````markdown
# Status M89 — Stage 1 source provisioning and binding contract

**State:** Complete. M89.1–M89.5 are complete; the whole-milestone review passed in iteration 5.

**Provenance:** “Stage 1 review: Kinet, APItools, OpenUdon, UWS” (`stage1-review.md`, 2026-09-28), finding F2 (source/local P2), F3 (design decision; retain strict refusal), F4 (source/local Lower), and Kinet consumer findings K1/K2 (source P2; local OpenUdon impact P2 for the package-write and binding contract, with Kinet W04 as consumer). Review and revalidation baseline: OpenUdon `7eb8a1b8065392c41671a19398c8c4f3d86fbc7c`; worktree clean. The reviewed APItools baseline is `26bb05247d6c48f8ee60b9ae178f6ef6d48bbe3d`; Kinet baseline is `395ecb8761c296b4dbda12bd0114d1cc61568112`. The review reports clean sibling worktrees. Earlier M87/M88 remain completed and retired; M89 is a new Stage 1 remediation, not S2b.

| Item | State | Notes |
|---|---|---|
| M89.1 — Consume the corrected APItools contract | `[+]` | F2 and downstream of APItools M79. Pinned exact published APItools commit `e3625f6ef52ea54b7f78b7a4a4f1993bf8a06a46` / module `v0.0.0-20260928033144-e3625f6ef52e` in `go.mod` and `go.sum`; verified Go module lookup and OpenUdon consumer behavior in both workspace mode and standalone `GOWORK=off` mode. Focused selected-nullable and unrelated-nullable-sibling tests pass in both modes; the full suite against the exact standalone module also passed before the permanent pin. APItools owns parsing/materialization; OpenUdon owns its consumption. |
| M89.2 — Add a bounded local-source package-write command | `[+]` | K1, source/local P2. Added `openudon step source add` with strict bounded JSON, exact local-file digests, source-family/extension checks, symlink-free reads, APItools inventory validation, and an optimistic manifest revision. Source files are create-only; source plus deterministic `expected/api-source-manifest.json` install atomically under the workflow package. Handoff inventory includes the manifest. Results omit absolute input paths; rollback-indeterminate outcomes list package-relative paths and prohibit blind retry. Focused OpenUdon workspace and standalone `GOWORK=off` tests pass for source write, append, stale revisions/digests, invalid batches, symlinks, package inventory, CLI transport, and runtime JSON-schema conformance. |
| M89.3 — Support explicit nested and renamed field mappings | `[+]` | K2/F2, source/local P2. step bind no longer gates explicit mappings on APItools' name-based candidate match; it requires supported input/output source capabilities and validates each mapped source field against its nested contract path, type, format, and requiredness. step check accepts optional output_mappings from contract paths to exact response paths; omission preserves same-name behavior. Nested object request and response aliases pass, conflicts fail with contract-field JSON Pointers, and selected nullable outputs remain indeterminate. Kinet must retain confirmed output mappings outside UWS intent and resend them to later checks. Added example-nested-mapping plus request/result fixtures. Verification: focused OpenUdon mapping, step-authoring schema/CLI tests and git diff --check pass. |
| M89.4 — Keep unknown effects and unsupported root extensions fail-closed | `[+]` | F3 remains strict: a contract effect of write does not authorize an operation whose source effect is unknown; bind returns effect.unknown without writing. Root-level x-* extensions remain unsupported; bind and candidates name the exact extension key, Check returns its contract JSON Pointer, and none of these results echo extension values. Regression tests cover write/unknown effect, check, bind, candidates, and no-write behavior. Verification: focused OpenUdon mapping, step-authoring schema/CLI tests and git diff --check pass. |
| M89.5 — Qualify standalone and Kinet consumer use | `[+]` | The W04.6 empty-package consumer journey exposed that synthesis omitted `expected/api-source-manifest.json` from the review handoff even though package validation requires it. `reviewHandoffInputs` now includes and package-validates the regular provenance manifest. After published revision `3a05d794f086d3dcf52cdb0c4bc86ea19aea7795` passed the consumer journey, deep review found and fixed a symlinked-parent check gap with a regression test in `2e2ecedb32add53e10d6d81d4b2528ca2bdfdcc8`, published to `origin/main`. Full standalone Go tests, `make check`, sibling/boundary/doc-memory/UWS checks, and `git diff --check` pass. Kinet W04.6's clean exact-revision consumer check and focused nested-mapping test pass against published `2e2ecedb32add53e10d6d81d4b2528ca2bdfdcc8`; the saved review handoff now retains the generated manifest and runtime data artifact. |

**Acceptance.** APItools M79's exact accepted revision is consumed. Explicit local sources are validated and written only inside the workflow package with content provenance; Kinet confirmation precedes all package writes. Renamed/nested mappings preserve evidence. Unknown effects and unsupported root extensions remain fail-closed with actionable diagnostics. Run `go test ./...`, `go run ./cmd/openudon check`, `go run ./cmd/openudon check-apitools-boundary`, `(cd tabilet && go run ../cmd/openudon check-doc-memory)`, `go run ./cmd/openudon validate ./examples/uws-validation`, `make check`, and `git diff --check`; Kinet's consumer check passes on the accepted revision.

**Dependencies.** The upstream operation-metadata contract is owned by APItools; its accepted, published revision is a start gate in Kinet `docs/kinet-order.md`. Kinet's consuming workflow milestone must reconcile and use this package's exact accepted, published revision. This scope does not modify UWS or promote OpenUdon S2b/S3.

**M89.1 source pin.** APItools `e3625f6ef52ea54b7f78b7a4a4f1993bf8a06a46`, module `v0.0.0-20260928033144-e3625f6ef52e`, published on `origin/main` 2026-09-28.

**Deep review.** Iteration 1 found a P2: `step source add` exposed only the caller-selected manifest ID, while operation references use a separate path-derived ID. The result now includes `candidate_source_id`; schema, fixture, docs, and regression checks cover the distinction. The active dashboard's stale W03 consumer reference is corrected. Focused step-authoring/schema/CLI tests and the disposable KINET_HOME add/bind/check run passed after the fix. Iteration 2 found a lower-severity stale header that still said M89.1 was consuming APItools M79 after that row had completed; the header now states the actual M89.1–M89.4 completion and M89.5 publication gate. Iteration 3 re-reviewed the complete implementation, fixtures, docs, and memory-bank updates and found no P1/P2 issues. The review-fix gate passed after three iterations. Iteration 4 found a lower-severity path-confinement gap: handoff inventory checked the API manifest's final component as a regular file but did not reject a symlinked `expected/` parent before its digest was read. The later package validator rejected that parent, so package qualification still failed closed. The fix now calls package path validation and adds a symlinked-parent regression; focused `internal/synthesize` tests pass. Iteration 5 re-reviewed the full M89 implementation, package-write and path-confinement behavior, mapping and refusal contracts, tests, fixtures, documentation, and downstream Kinet use. It found no remaining P1/P2-or-higher issues or lower findings to carry; the review-fix gate passed. The exact Kinet consumer run passed against clean published OpenUdon revision `2e2ecedb32add53e10d6d81d4b2528ca2bdfdcc8`.
````
