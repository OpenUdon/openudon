# Status E05 - Realistic Playwright-Browser Journey Qualification

## Goal

Qualify realistic local read and write workflows across Browsertools guided
authoring, strict OpenUdon import and synthesis, UWS 1.8, Udon v3, and real
Browserdriver Chromium replay.

## State

Complete.

## Dependencies

- UWS `dd9eb32105131bdbc2855090ae0639b22d12de2b`
  (`v0.0.0-20260817013720-dd9eb3210513`).
- Browserdriver `0efd276ad77cae23dd1c3bd915ea107890ec7204`.
- Udon `2d2f4979d9a66f5bcb723ab4944e17f31dd1a8b8`.
- Browsertools `4d940eaaae16390dd79b65f1829d66e198095d7d`
  (`v0.0.0-20260817224213-4d940eaaae16`).

## Task Ledger

| Item | State | Notes |
|---|---|---|
| Implement and qualify the realistic browser journey release suite | `[+]` | OpenUdon `5a825a41b298b4cb589756adbdef812573fc7522` adds eight strict `openudon.browser-journey.v1` cases, deterministic Browsertools guided bundles, strict canonical-profile re-import, ordered parameterized UWS 1.8 synthesis, real headless Udon/Browserdriver v3 replay, exact read/write/rejection/session postconditions, the value-free `openudon.browser-journey-eval.v1` report, CLI/Make/docs, and required local/tag release gates. |
| Correct the recorded implementation provenance | `[+]` | Replaced a manually expanded short hash with the exact `git rev-parse HEAD` value proven by the clean post-commit 8/8 journey report. No implementation or contract changed. |

## Scenario Matrix

| Case | Contract |
|---|---|
| `catalog-search-filter` | Text, radio, checkbox, select, submit navigation, and exact result outputs. |
| `catalog-pagination` | Two ordered operations preserve one named session and return page-two output. |
| `order-structured-read` | Exact accessibility, JSON-LD, microdata, and reviewed CSS fallback values. |
| `record-update-approved` | Exact runtime operation approval permits one POST and the reviewed final record state. |
| `record-update-unapproved` | Missing operation approval rejects before the POST. |
| `record-update-ambiguous` | Duplicate exact Save locators reject without a POST. |
| `parameter-contract-rejected` | Missing, undeclared, wrong-type, and origin-escape values fail closed. |
| `session-lifecycle` | Operations share one session within a run, while two executions receive distinct fresh runs. |

## Verification

- The real journey run and independent report verification passed 8/8 with no
  failure, skip, or quarantine.
- Full `go test ./...`, `go vet ./...`, `make check`, doc-memory validation,
  strict MkDocs, release/public workflow `actionlint`, and `git diff --check`
  passed.
- The legacy headed `password-main` loopback and its report verification passed
  after applying the release workflow's temporary AppArmor user-namespace
  setting; the host setting was restored to `1` afterward.
