# Status E04 - Complementary Real-Browser Scenario Evaluation

## Goal

Qualify the production Playwright-browser workflow with a deterministic release
suite and detect public-site drift with a separate explicit-network canary
suite.

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
| Implement, publish, and qualify both real-browser suites | `[+]` | OpenUdon `0ea9f3eff281cffbcd699b2b62905090b51d5e28` adds strict embedded manifests/lock/reporting, a 21-case required loopback from Browsertools author-session v2 through Udon/Browserdriver v3, four explicit-network anonymous presence canaries, CLI/Make/docs, and required/informational workflows. The clean coordinated runs passed 21/21 and 4/4 with no skipped or quarantined case. |
| Close post-publication review findings | `[+]` | OpenUdon `0a14a9fad7c86318f5d23028496963b4cfe01dcf` provisions sandbox-compatible user namespaces in both hosted Ubuntu jobs and rejects duplicate keys at every JSON object depth before wire decoding. Browserdriver `0efd276ad77cae23dd1c3bd915ea107890ec7204` makes only literal `presence: true` select Boolean match mode, so the immutable browser 1.7 schema's accepted `presence: false` preserves its declared type. |

## Protocol And Profile Matrix

| Path | Browsertools | Authentication | Capability | UWS | Driver |
|---|---|---|---|---|---|
| Loopback author/replay | author-session/result v2 | 1.1 | browser 1.5/1.6/1.7 | 1.8/1.9 as oldest sufficient | v3 |
| Public presence canary | live-check v1 | 1.0, credential-free | browser 1.5 presence | 1.7 | v2 |

## Verification

- `xvfb-run -a make release-saas-check` passed, including 21/21 required
  loopback cases, the 11-gate browser-free matrix, full Go tests/vet, strict
  docs, UWS validation, scorecards, and dry-run handoffs.
- The independent public run passed 4/4 fixed targets with explicit network
  authority and no credential/provider inputs.
- The clean review-repair matrix passed 21/21 loopback and 4/4 public cases at
  OpenUdon `0a14a9f` and Browserdriver `0efd276`; every repository dirty bit
  was false. The workflow-equivalent Ubuntu sysctl preparation launched
  sandboxed Chromium successfully and restored the local AppArmor restriction.
- `go test -race ./internal/browserscenario ./internal/icot`, `actionlint`,
  strict MkDocs, report digest verification, and `git diff --check` passed.
