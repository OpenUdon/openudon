# Retired milestone M34 - M34 - Eval Seed/Build Matrix

**Milestone.** M34
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M34.md
**Source status SHA-256.** 06ced660c33c623d4c361e9602329ce496e529770fc194f904ff302a5ea8b31f
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
# M34 - Eval Seed/Build Matrix

## Goal

Make the curated `examples/eval/*` corpus executable as a deterministic seed/build matrix. Each
fixture declares whether `icot --from-example` plus `openudon build` must pass or is expected to
fail for a deliberate negative-policy reason.

## Status

| Item | State | Notes |
|---|---|---|
| Matrix policy schema | `[+]` | `reference/policy.json` accepts `seed_build.expected`, `class`, `reason`, and optional `allowed_failure_codes`. |
| Fixture classification | `[+]` | All eval fixtures now declare explicit seed/build expectations. Strict positives and advisory reducibility fixtures are expected to pass; disallowed runtime remains an expected negative failure. |
| Deterministic harness | `[+]` | `TestEvalReferenceSeedBuildMatrix` materializes each fixture through iCoT into a temp package and builds from `intent.hcl` without network retrieval or LLM calls. |
| Hermetic fixture discovery | `[+]` | Commit `1cb5a54` centralizes policy-backed fixture discovery for the scorecard and matrix, ignores bare/sketch directories without `reference/policy.json`, fails the matrix when no conforming fixture exists, and keeps the full suite passing with the ignored `mucker-next-steps-agent` sketch present. |
| Strict fixture remediation | `[+]` | Fixed source security binding normalization, delivery dependency cleanup, literal `path` parameter lowering, and webhook-validation side-effect classification. |
| Documentation | `[+]` | Added `docs/eval-seed-build-matrix.md` and linked the eval gallery to the seed/build policy. |
| Verification | `[+]` | `make eval-seed-build`, focused package tests, `go test ./...`, `go vet ./...`, `make check`, `openudon validate`, strict MkDocs build, doc-memory, boundary, sibling, and diff checks pass locally. |

## Boundary Notes

- The matrix is provider-free: it uses reference intent and package-local API source artifacts only.
- Advisory fixtures remain separate from strict OpenUdon-native behavior even when they build cleanly.
- Strict positive fixtures must not rely on advisory classification or failure allowances.
~~~~~~~~~~~~~~~~~~~~
