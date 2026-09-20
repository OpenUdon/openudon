# Status M53 - Shared Authoring iCoT Consumption

State of OpenUdon consuming the shared Authoring interactive iCoT APIs.

## Goal

Keep `go run ./cmd/icot` behavior and artifacts stable while replacing
duplicated generic progressive-loop plumbing with
`github.com/OpenUdon/authoring/icot`.

## Tasks

| Item | State | Notes |
|---|---|---|
| M53.1 loop delegation | `[+]` | `internal/authoring` now aliases/delegates prompt sessions, events, extractor hooks, progressive hooks, and lifecycle execution to public Authoring iCoT APIs. |
| M53.2 downstream ownership | `[+]` | OpenUdon still owns project.md, intent.hcl, prompts, workflow intent schema, catalog planning, repair/review behavior, reports, scorecards, and package artifacts. |
| M53.3 compatibility tests | `[+]` | Focused internal authoring, elicitor, iCoT, and cmd/icot tests pass with the shared loop. |
| M53.4 no-LLM/manual loop migration follow-up | `[+]` | `elicitor.Run` now routes no-LLM/manual and verify-only runs through the shared progressive path. Verify-only disables drafting and skips draft repair/model review side effects after render validation; the bespoke manual authoring driver was removed while downstream final edit/confirmation helpers remain. |
| M53.5 decision evidence adapter binding | `[+]` | OpenUdon decision evidence now merges through `authoring/decision` for normalization, duplicate merge, confidence behavior, and conflict marking while preserving OpenUdon JSON/YAML `reason` compatibility and multi-value credential decisions. |
| M53.6 scorecard base-contract binding | `[+]` | Scorecard writing embeds and validates an `authoring.scorecard.v1` view while retaining `openudon.icot-scorecard.v1` extension counters for provider families, false passes, diagnostic gaps, and eval policy. |

## Acceptance Criteria

- `go run ./cmd/icot` remains the runnable OpenUdon iCoT command.
- OpenUdon-specific artifact layout and report/version semantics stay in
  OpenUdon.
- Authoring owns only generic loop mechanics and CLI plumbing; provider clients
  remain downstream.

## Verification

- `go test ./internal/authoring ./internal/icot ./internal/icot/elicitor ./cmd/icot`
- Broader `go test ./...` and `go vet ./...` are still recommended before
  publishing a module bump.
