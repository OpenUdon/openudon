# A22 Status - Operator-Authored Content Trust

State markers and commit rules are defined in [milestone.md](milestone.md).

## State

Completed at OpenUdon `51de359` after published UWS 1.9.1
`9e676eaa469e` and Browsertools M28 `75fd5c3ab81f`. OpenUdon's local baseline
and A22 commits were qualification inputs, not publication authority. P06 and
E12 subsequently completed through OpenUdon `cc378be`; the user later
authorized ordered publication, and OpenUdon is published through hosted-CI
follow-up `2c99fde`.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| A22.1 Pin published contracts and add strict intent declarations | `[+]` | OpenUdon `701cd15` pins published UWS `9e676eaa469e` and Browsertools `75fd5c3ab81f`; adds operator-facing `content_trust` HCL/JSON types for package-relative source descriptions, operation defaults/outputs, triggers, and workflow defaults/inputs; and validates exact levels, no-op objects, duplicates, and unresolved intent references. Parse/render/clone round trips are deterministic. Focused workflow-intent/synthesis tests, vet, and diff checks pass. Task review iteration 1 found no P1/P2-or-higher issue. |
| A22.2 Lower declarations and select UWS 1.9.1 conditionally | `[+]` | OpenUdon `4110185` resolves package source paths to generated source-description IDs, lowers sanitized leaf operation IDs plus trigger and `main` workflow declarations, materializes workflow input schemas only when per-input trust is present, and selects UWS 1.9.1 only for a non-nil registry. Existing browser 1.7 generation still asserts UWS 1.9.0. Deterministic YAML/HCL round trips and undeclared-output/no-op failures pass focused synthesis/workflow-intent tests and vet. Task review iteration 1 found no P1/P2-or-higher issue. |
| A22.3 Align authoring documentation and close A22 | `[+]` | OpenUdon `51de359` documents the operator-only provenance contract, conditional 1.9.1 selection, and non-authorization boundary; fixes the iteration-1 workflow-default input-schema finding; and adds its regression. Full workspace and standalone tests/vet, `make check`, strict MkDocs, focused HCL-doc parsing, and diff checks pass. The public authoring/review direction triggers evolution prompt v33; its result remains absent until P06 and E12 pass. Milestone review iteration 2 found no remaining P1/P2-or-higher issue. |

## Compatibility And Non-Goals

- Content trust is review metadata, not a credential, approval, policy, or
  runtime-enforcement mechanism.
- Existing intent without declarations and existing UWS 1.9.0/browser 1.7
  packages remain unchanged.
- A22 does not run the analyzer, change quality status, authorize execution,
  publish commits, or advance W8M.

## Review Log

- Milestone review iteration 1 found one P2: a workflow default declaration
  with authored inputs materialized the UWS input schema only when a per-input
  override was also present. A22.3 must materialize all authored entry inputs
  whenever `main` has any trust declaration, then rerun focused and full gates.
- Milestone review iteration 2 verified the correction, conditional version
  gate, intent/wire reference integrity, legacy browser 1.7 behavior,
  operator-only authoring boundary, documentation, and complete gates. No
  P1/P2-or-higher finding remains.

## Scoped Commits

- Tofu milestone opening: `075def9`.
- OpenUdon A22.1: `701cd15`.
- OpenUdon A22.2: `4110185`.
- OpenUdon A22.3 and review correction: `51de359`.

## Verification

- Focused `internal/workflowintent` and `internal/synthesize` tests.
- `go test ./...` and `go vet ./...`.
- `GOWORK=off go test ./...` and `GOWORK=off go vet ./...`.
- `make check`, strict documentation build where configured, and
  `git diff --check` in OpenUdon and Tofu.
- Bounded deep-review gate with no unresolved P1/P2-or-higher finding.
