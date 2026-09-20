# Status A11 - Phase C Real-Browser Qualification

## Goal

Make the interactive Phase C lifecycle a required provider-free release
property exercised by real Chromium against the production loopback handler.

## State

Complete.

## Dependencies

- A10 interactive embedded authoring and review shell.
- Existing release-runner Chromium installation and sandbox-compatible user
  namespace setup.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| A11.1 test-only real-browser harness | `[+]` | A build-tagged Playwright-Go v0.6201.0 suite starts the actual handler on an ephemeral loopback listener and uses the real capability bootstrap/cookie flow. Production UI packages do not import Playwright. |
| A11.2 accessibility and authoring journeys | `[+]` | Chromium coverage proves unique accessible names, skip-link and frontier keyboard order, visible focus, recommendation fill, required complete-round validation, exact submitted answers, preview/action/conflict rendering, and review/overwrite gates. |
| A11.3 approval and failure lifecycle | `[+]` | Browser journeys prove explicit final and incomplete flags, frozen completion, stale unsent-input preservation and adoption, restart-required drift, editable domain rejection, reconciled explicit retry, indeterminate lockout, and discarded old snapshot responses delivered after rejected or successful mutations. |
| A11.4 polling and responsive behavior | `[+]` | A controlled browser clock proves visible two-second polling, 2/4/8/16/30-second error backoff, hidden pause, immediate visibility refresh, and unchanged `304` handling. Viewport and zoom checks prove no page-level horizontal overflow at 360 CSS pixels or 200 percent. |
| A11.5 required release integration | `[+]` | `make icot-ui-browser-check` discovers and runs all 13 build-tagged `TestPhaseCBrowser*` journeys, rejects `OPENUDON_ICOT_UI_BROWSER_DISABLE_SANDBOX=1`, and is required by `make release-saas-check` and tag automation. On 2026-08-21 the release qualification passed under Xvfb with sandbox-compatible user namespaces; every journey logged `chromium_sandbox_enabled=true sandbox_required=true`, and the host setting was restored afterward. |
| A11.6 qualification documentation | `[+]` | README, operator and release docs, tech-stack/architecture/milestone memory, and evolution v25 distinguish the required test-only browser harness from the browser-free production runtime. |

## Acceptance Criteria

- Real-browser failures block the provider-free release gate.
- The default and release paths keep Chromium sandboxing enabled.
- The harness uses no external network, provider credentials, workflow
  execution, browser evidence retention, or production browser dependency.

## Verification

- `make icot-ui-browser-check` passed with sandboxed Chromium and without the
  disable override; restricted hosts may still use the separately named
  diagnostic target, but it cannot satisfy release qualification.
- Full standalone, race, vet, repository, documentation, cross-build, and
  scorecard gates pass.
