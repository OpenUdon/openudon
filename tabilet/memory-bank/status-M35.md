# M35 - v0.1.2 Release Readiness

## Goal

Prepare the OpenUdon v0.1.2 public release surface now that eval seed/build coverage is explicit and
green. Keep the scope on release evidence, docs, package hygiene, and compatibility gates before
moving on to v0.1.3 AsyncAPI or v0.1.4 browser/vendor-profile artifacts.

## Status

| Item | State | Notes |
|---|---|---|
| Release gate inventory | `[+]` | v0.1.2 provider-free evidence is `go test ./...`, `go vet ./...`, `make check`, `make release-check`, `make eval-seed-build`, `make release-saas-check`, UWS validation, doc-memory, n8n bridge validation, strict MkDocs, and `git diff --check`. `make release-saas-check` now includes `eval-seed-build`. |
| Request mapping diagnostics | `[+]` | Missing request fields now report source path, operationId, missing field, and known request fields. Missing operationIds report available operationIds from the selected source. Ambiguous request aliases tell operators to use `path.<name>`, `query.<name>`, `header.<name>`, or `body.<name>`. |
| Dry-run handoff lane | `[+]` | M35 automated evidence stays provider-free through `openudon run --dry-run` demo packages. Weather-to-Gmail real udon/provider execution is documented as manual, credential/env-owned, and side-effectful. |
| Advisory fixture posture | `[+]` | All n8n advisory reducibility fixtures now seed and build from bounded package-local OpenAPI evidence while remaining advisory rather than strict-native. |
| Release docs | `[+]` | README, release stewardship, SaaS operator release path, and weather tutorial now describe the eval seed/build matrix, comprehensive provider-free gate contents, strict MkDocs, n8n bridge validation, and dry-run-only release demos. |
| Package hygiene | `[+]` | `examples/support-priority-routing/` is ignored local generated scratch output, not a promoted top-level example. Release demo output remains under ignored `.openudon-run/`. |
| Cross-repo compatibility | `[+]` | `go run ./cmd/openudon check` found required sibling repositories. `go run ./cmd/openudon check-apitools-boundary` passed. Observed sibling heads: `../uws` `d922781` with unrelated docs/mkdocs local changes, `../apitools` `3df8f5e` clean, `../udon` `0404b29` clean. No private udon import or public OpenUdon boundary failure was observed. |
| Final release evidence | `[+]` | Passed locally: focused request mapping diagnostics tests; `go test ./...`; `go vet ./...`; `make eval-seed-build`; `make check`; UWS validation; doc-memory; n8n bridge validation; `mkdocs build --strict --site-dir /tmp/openudon-mkdocs-m35`; and `make release-saas-check`, including Gmail audit receipt and order fulfillment sandbox dry-run demos. |

## Boundary Notes

- v0.1.2 remains OpenAPI/UWS authoring, review, package, and executor-handoff focused.
- Real udon/provider execution stays manual and is not a v0.1.2 release gate.
- AsyncAPI stays deferred to v0.1.3.
- Browser/vendor-profile artifacts stay deferred to v0.1.4.
