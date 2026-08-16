# Browser Integration Evaluation

`openudon browser-integration-eval` is the deterministic release-evidence
matrix for anonymous and authenticated browser authoring through trusted replay. It runs named,
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
synthetic records and fake browser implementations, exercises iCoT's strict
live protocol/result adapters without launching a child browser, checks that
iCoT has no Browsertools capture or Playwright implementation dependency, runs
Browserdriver's offline v2/v3 protocol tests, and uses Browsertools doctor only to
observe pinned component availability without installation, browser launch, or
network access.

## Required Matrix

| Gate | Evidence |
|---|---|
| OpenUdon authoring | API preference, anonymous handoff, explicit authenticated live orchestration, disclosure denial/human fallback, minimal child environment, private result digest validation, atomic staging, and malformed/tampered rejection |
| OpenUdon package/handoff | Strict live and portability verification, private/tampered input rejection, value-free package review, authentication/capability separation, UWS 1.7/1.8 discriminator selection, and trusted dry-run |
| iCoT dependency boundary | The `cmd/icot` dependency graph contains no Browsertools capture, Playwright adapter, or Playwright-Go implementation package |
| OpenUdon repository boundary | Production source contains no private executor, desired-state parser, or removed apitools lifecycle imports |
| Browsertools producer | Fake author-session transitions, human credential/MFA checkpoints, reduced observations, origin/action/POST gates, deterministic profile synthesis, context bounds, and offline doctor behavior |
| UWS contract | Immutable browser 1.5/authentication 1.0 compatibility plus UWS 1.8 browser 1.6/authentication 1.1/call 1.1 contexts, dispatch, round trips, and rejection fixtures |
| Udon consumer | Private source loading, runtime approvals, authentication, opaque sessions, v2 compatibility, v3 context replay, and explicit driver policy |
| Browserdriver runtime | Offline v2/v3 NDJSON, exact-origin and child-context guards, popup/frame inventory, ambiguity rejection, credential lookup, and session isolation |
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
`--headed-auth` separately runs both the headed authentication fixture and the
same-context redirect-login author-session fixture. If
a requested pinned driver or browser is unavailable, the corresponding result
is recorded honestly as `skipped`; the evaluator never installs it. These
flags do not authorize a real website, account, credential, MFA challenge, or
production side effect.

The generated report is release evidence, not runtime authority. Keep any
private Browsertools cache, guided evidence, assisted-authentication bundle,
raw/rich capture, cookie, storage state, or live session outside OpenUdon
packages and outside this report.
