# Retired milestone M73 - Status M73 - Shared Interview Settlement And Lifecycle-Ranking Boundary

**Milestone.** M73
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M73.md
**Source status SHA-256.** 9df53c4c779ac7ee256eab41b36aa049b0fda709aed4bbeda922062f8a82006a
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
# Status M73 - Shared Interview Settlement And Lifecycle-Ranking Boundary

State of OpenUdon's adoption of the coordinated Authoring interview contract
and Apitools lifecycle-ranking ownership correction.

## Goal

Use one generic complete-frontier transaction across OpenUdon entry points and
move prompt-safe lifecycle-role ranking to the API metadata module without
moving workflow policy out of OpenUdon.

## Task Status

| Item | State | Notes |
|---|---:|---|
| Shared interview binding | `[+]` | The elicitor delegates projection and settlement to `icot.InterviewBinding`; OpenUdon callbacks retain workflow preparation, questions, product mutation, normalization, and validation. |
| Runtime integration | `[+]` | Progressive execution supplies the same binding and keeps OpenUdon-only post-round behavior in the engine's post-settlement hook. |
| Lifecycle package migration | `[+]` | OpenUdon consumes `apitools/operationlifecycle` through source-bearing `apitools.OperationSummary` values; no Authoring compatibility import remains. |
| Source identity | `[+]` | Ranked candidates map back through source ID and operation ID so duplicate IDs cannot cross documents. |
| Product wording | `[+]` | OpenUdon explicitly retains the `Workflow goal` opening label while generic Authoring defaults remain product-neutral. |
| Workspace verification | `[+]` | Full tests, vet, the iCoT authoring scorecard, and Authoring compatibility checks pass in the sibling workspace. |
| Public dependency pins | `[+]` | OpenUdon `065bb84` pins published Apitools `v0.0.0-20260820042238-d51b61ead067` and Authoring `v0.0.0-20260820042256-2f73e3526583`; standalone test/vet pass without a local replacement. |

## Scoped Commits

- Apitools lifecycle ranking: `d51b61e`
- Authoring interview/persistence/boundary series: `243121d`, `eee0c0d`, `2f73e35`
- OpenUdon adapter migration: `43294bf`
- OpenUdon public dependency pins: `065bb84`
- Ramen adapter migration: `330fd89`
- Ramen public dependency pins: `721b540`

## Verification

- `go test ./...`
- `go vet ./...`
- `make icot-authoring-scorecard`
- `(cd ../authoring && ./scripts/check-compat.sh)`
- `go run ./cmd/openudon check-doc-memory`
- `git diff --check`
- `git -C ../tofu diff --check -- openudon`

Standalone `GOWORK=off go test ./...` and `GOWORK=off go vet ./...` pass with
the published Apitools and Authoring pins.
~~~~~~~~~~~~~~~~~~~~
