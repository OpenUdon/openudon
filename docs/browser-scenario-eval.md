# Browser Scenario Evaluation

`openudon browser-scenario-eval` owns two complementary real-browser suites.
They test the portable workflow boundary without storing a DOM, accessibility
snapshot, screenshot, page value, credential, cookie, browser state, or child
process output.

| Suite | Purpose | Authority | Release posture |
|---|---|---|---|
| `loopback` | Deterministic Browsertools author-session v2 through OpenUdon staging, UWS synthesis, Udon v3 lowering, and Browserdriver v3 replay | Local random-port HTTP only; headed Chromium; synthetic credential values remain inside trusted replay | Required real-browser release gate |
| `public` | Detect external markup, accessibility-name, resource-origin, and runtime drift | Explicit `--allow-network`; four fixed anonymous HTTPS targets; headless read-only presence checks | Weekly/manual informational canary |

The separate [Browser Integration Evaluation](browser-integration-eval.md)
remains the fast browser-free contract matrix. Ordinary `go test`, `make
check`, and `make release-check` do not launch or install a browser.

## Run The Deterministic Loopback Suite

Install the pinned Browsertools Playwright-Go Chromium and Browserdriver Node
Playwright Chromium first. On a desktop with a display:

```bash
make browser-scenario-loopback
```

On headless Linux:

```bash
xvfb-run -a make browser-scenario-loopback
```

The 21 embedded cases cover password-only authentication, all eight reviewed
MFA kinds, main/popup/frame contexts, exact-name and unique-role locators,
zero/16/17 outputs, typed string/integer/number/Boolean/presence results,
noncanonical scalar rejection, stale and ambiguous targets, context
substitution, origin escape, and secret-output rejection. Every case starts a
fresh loopback server, private authoring root, browser context, Udon workdir,
and Browserdriver lifecycle; teardown is part of the result.

Run a bounded subset by repeating `--scenario`:

```bash
go run ./cmd/openudon browser-scenario-eval \
  --suite loopback \
  --scenario mfa-totp-scalars \
  --scenario popup-context \
  --require-ready \
  --out eval/runs/browser-scenario-loopback-local/report.json
```

Without `--require-ready`, a missing installed browser dependency is recorded
as `skipped`; release automation always requires readiness.

## Run The Public Canaries

Public execution is never implicit:

```bash
make browser-scenario-public

# Equivalent direct form:
go run ./cmd/openudon browser-scenario-eval \
  --suite public \
  --allow-network \
  --require-ready \
  --out eval/runs/browser-scenario-public-local/report.json
```

The embedded inventory is Books to Scrape, Hacker News, the JavaScript Quotes
to Scrape page, and the Wikipedia Playwright article. Each manifest fixes the
target URL, exact resource-origin allowlist, accessibility role/name probe,
and an exact-one match bound. Browsertools first produces a strict value-free
`browsertools.live-check.v1` report. Udon and Browserdriver then independently
establish a credential-free ephemeral v2 session and return only Boolean
presence outputs from a browser 1.5 profile carried by UWS 1.7. No account,
form submission, credential, MFA challenge, mutation, or production side
effect is involved.

Public failures use only closed classes such as `target_unreachable`,
`timeout`, `origin_policy_drift`, `shape_drift`, and `contract_drift`. A
maintainer may add a manifest quarantine only for a documented upstream reason,
with fixed start/end dates no more than 14 days apart. Quarantine is visible in
the report and cannot silently become a pass.

## Report And Compatibility Contract

Both suites write `openudon.browser-scenario-eval.v1` JSON and an adjacent
`.sha256` sidecar. The report contains exact repository commits, public module
versions, closed phase/assertion/detail identifiers, counters, and explicit
safety booleans. It contains no target page content or subprocess output and
is safe to archive. Verify it independently:

```bash
go run ./cmd/openudon browser-scenario-eval \
  --verify eval/runs/browser-scenario-loopback-local/report.json
```

The embedded `openudon.browser-scenario-lock.v1` fixes Browsertools, Udon,
Browserdriver, UWS, Go, Node, and Playwright compatibility. Execution rejects
a sibling checkout or OpenUdon module pin that differs from the lock. Scenario
manifests and reports strict-decode unknown or duplicate fields and apply
finite bounds before any browser or network authority is exercised.

## Where The V2 Contract Is Documented

OpenUdon's client and review flow are specified in [Authenticated
Goal-Directed Browser Authoring](authenticated-browser-authoring.md), especially
the “Human And Model Checkpoints” and “Local Protocol And Artifacts” sections.
The canonical producer-side message/state contract is Browsertools'
[Authenticated goal-directed browser authoring](https://github.com/OpenUdon/browsertools/blob/main/docs/authenticated-goal-authoring.md).
Version 2 means `browsertools.author-session.v2` input,
`browsertools.authenticated-authoring.v2` results, reviewed human-only MFA
kinds, and up to 16 reviewed typed outputs. Protocol v1 is rejected without a
fallback.

Downloads, cookie transfer, goal inference, selectors, arbitrary scripting,
credential export, and browser-state reuse remain outside both scenario suites.
