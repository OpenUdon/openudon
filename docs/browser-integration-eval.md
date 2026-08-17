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
| OpenUdon authoring | API preference, anonymous handoff, strict author-session v2 orchestration, human-only typed MFA/output review, disclosure denial/human fallback, minimal child environment, exact bounds/context authority, a real Browsertools-produced private result through validation/staging, and malformed/tampered/substituted rejection |
| OpenUdon package/handoff | Strict live and portability verification, private/tampered input rejection, value-free package review, authentication/capability separation, UWS 1.7/1.8/1.9 discriminator selection, and trusted dry-run |
| iCoT dependency boundary | The `cmd/icot` dependency graph contains no Browsertools capture, Playwright adapter, or Playwright-Go implementation package |
| OpenUdon repository boundary | Production source contains no private executor, desired-state parser, or removed apitools lifecycle imports |
| Browsertools producer | Observation-generation authority, human-selected MFA kind, bounded reviewed outputs, action-time exact-name/unique-role proof, complete context inventory, current goal proof, deterministic output, and offline doctor behavior |
| UWS contract | Immutable older compatibility plus UWS 1.9/browser 1.7 scalar accessibility conversion, context contracts, fresh union decoding, dispatch, round trips, and rejection fixtures |
| Udon consumer | Private source loading, runtime approvals, authentication, opaque sessions, v2 rejection/v3 browser 1.7 replay, scalar post-conversion validation, redaction rejection, and real producer pairs through output validation |
| Browserdriver runtime | Offline v2/v3 NDJSON, browser 1.7 v3-only strict scalar conversion, failure non-disclosure, exact-origin/context guards, ambiguity rejection, credential lookup, and session isolation |
| Component inventory | Browsertools doctor reports pinned Chromium, Firefox, and WebKit readiness without installing or launching anything |

The report contract is `openudon.browser-integration-eval.v1`. Validation fixes
the gate order, repository names, command argv, assertions, authority claims,
counter totals, and closed value-free detail vocabulary. Passing Go gates must
contain every named test marker, not merely an overall package success. Reports
also require the producer-to-consumer and producer-to-replay test names; a
hand-built compatible fixture does not establish either seam. Browserdriver's
npm gate similarly requires the named v3 replay and cached-context freshness
tests rather than inferring coverage from a passing-test count. Reports
record the commit and dirty-worktree bit for every participating repository,
are written atomically under ignored `eval/runs/`, never include repository
paths or captured command output, and can be verified only with their matching
digest sidecar. Verification rejects a structurally valid report whose matrix
status is `fail` when used as a release gate.

The Go module pins name the exact Browsertools and UWS feature commits used by
the seam tests. During a coordinated pre-publication review, standalone tests
may use process-local Git URL mappings to clean local clones of those exact
commits. Release evidence must use ordinary module resolution after the commits
are published; an unreachable pseudo-version or a committed `replace` is not a
releasable pin.

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
