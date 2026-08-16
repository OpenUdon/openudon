# Browser Integration Evaluation

`openudon browser-integration-eval` is the deterministic release-evidence
matrix for the browser authoring-to-handoff boundary. It runs named,
provider-free checks in sibling OpenUdon, Browsertools, UWS, Udon, and
Browserdriver checkouts and writes one value-free report with an adjacent
SHA-256 sidecar.

```bash
make browser-integration-check

# Equivalent direct commands:
go run ./cmd/openudon browser-integration-eval \
  --out eval/runs/browser-integration-local/report.json
go run ./cmd/openudon browser-integration-eval \
  --verify eval/runs/browser-integration-local/report.json
```

The default run does not launch a browser, contact a target, read credential
values, execute a workflow, or retain subprocess stdout/stderr. It exercises
synthetic records and fake browser implementations, checks that iCoT has no
Browsertools capture or Playwright implementation dependency, runs
Browserdriver's offline protocol tests, and uses Browsertools doctor only to
observe pinned component availability without installation, browser launch, or
network access.

## Required Matrix

| Gate | Evidence |
|---|---|
| OpenUdon authoring | API preference, anonymous handoff, login-required failure, private-root containment, guided-result replay/deduplication/bounds, and secret, literal-value, stale, malformed, or decision-mismatch rejection |
| OpenUdon package/handoff | Strict live and portability verification, private/tampered input rejection, value-free package review, authentication/capability separation, and trusted dry-run |
| iCoT dependency boundary | The `cmd/icot` dependency graph contains no Browsertools capture, Playwright adapter, or Playwright-Go implementation package |
| OpenUdon repository boundary | Production source contains no private executor, desired-state parser, or removed apitools lifecycle imports |
| Browsertools producer | Fake-engine capture/check behavior, guided and assisted-authentication closure, value-free report wires, and offline doctor behavior |
| UWS contract | Existing browser 1.5 and browser-authentication schemas, source bindings, and named-session extension behavior remain unchanged |
| Udon consumer | Private source loading, runtime approvals, authentication, opaque sessions, persistent driver protocol, and explicit driver policy |
| Browserdriver runtime | Offline NDJSON protocol, exact-origin, credential lookup, and session-isolation tests |
| Component inventory | Browsertools doctor reports pinned Chromium, Firefox, and WebKit readiness without installing or launching anything |

The report contract is `openudon.browser-integration-eval.v1`. Validation fixes
the gate order, repository names, command argv, assertions, authority claims,
counter totals, and closed value-free detail vocabulary. Passing Go gates must
contain every named test marker, not merely an overall package success. Reports
record the commit and dirty-worktree bit for every participating repository,
are written atomically under ignored `eval/runs/`, never include repository
paths or captured command output, and can be verified only with their matching
digest sidecar. Verification rejects a structurally valid report whose matrix
status is `fail` when used as a release gate.

## Installed Browser Opt-Ins

Installed browsers are not a prerequisite for the default release gate. To
request separate loopback-only evidence:

```bash
go run ./cmd/openudon browser-integration-eval \
  --installed-engines \
  --out eval/runs/browser-integration-installed/report.json

go run ./cmd/openudon browser-integration-eval \
  --installed-engines --headed-auth \
  --out eval/runs/browser-integration-headed/report.json
```

`--installed-engines` runs the existing Chromium live/rich checks and the
Chromium/Firefox/WebKit portability check against local loopback fixtures.
`--headed-auth` separately runs the headed authentication loopback fixture. If
a requested pinned driver or browser is unavailable, the corresponding result
is recorded honestly as `skipped`; the evaluator never installs it. These
flags do not authorize a real website, account, credential, MFA challenge, or
production side effect.

The generated report is release evidence, not runtime authority. Keep any
private Browsertools cache, guided evidence, assisted-authentication bundle,
raw/rich capture, cookie, storage state, or live session outside OpenUdon
packages and outside this report.
