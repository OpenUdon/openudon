# Browser Scenario Evaluation

`openudon browser-scenario-eval` owns three complementary real-browser suites.
They test the portable workflow boundary without storing a DOM, accessibility
snapshot, screenshot, page value, credential, cookie, browser state, or child
process output.

| Suite | Purpose | Authority | Release posture |
|---|---|---|---|
| `loopback` | Deterministic Browsertools author-session v2 through OpenUdon staging and trusted replay | Local random-port HTTP only; headed Chromium; synthetic credential values remain inside trusted replay | Required current-stack real-browser release gate |
| `journey` | Reviewed read/write workflows plus Browser 1.8/1.9 template and mixed-session replay | Local random-port HTTP only; headless Chromium; fixture state is inspected after replay | Required current-stack real-browser release gate |
| `public` | Detect external markup, accessibility-name, resource-origin, and runtime drift | Explicit `--allow-network`; four fixed anonymous HTTPS targets; headless read-only presence checks | Weekly/manual informational canary |

The separate [Browser Integration Evaluation](browser-integration-eval.md)
remains the fast browser-free contract matrix. Ordinary `go test`, `make
check`, and `make release-check` do not launch or install a browser.

`make browser-scenario-loopback`, `make browser-scenario-journey`, and the
hosted release workflow explicitly select `--stack current`. The CLI defaults
to `--stack historical` for existing callers and local qualification. Public
canaries remain historical and require their separate network opt-in. The
historical compatibility lock and v1 report readers are unchanged.

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

For a clean Browserdriver checkout without `node_modules`, set
`OPENUDON_BROWSERDRIVER_NODE_MODULES` to the separately prepared, lock-matched
directory when invoking either local Make target.

Chromium still runs with its sandbox enabled. Ubuntu 24.04 hosted runners also
need unprivileged user namespaces: the release and public-canary workflows
explicitly enable `kernel.unprivileged_userns_clone` and, when present, disable
the AppArmor-only `kernel.apparmor_restrict_unprivileged_userns` restriction on
their ephemeral runner before launch. They fail if either requested setting
does not take effect; the suites never add `--no-sandbox`.

The 23 embedded cases use session-gated goal pages and server-verified
credential/MFA challenges. For push, number-match, passkey, and security-key,
the evaluator waits for Udon's exact structured terminal prompt, latches
approval to the one pending observed server session, and only then supplies
`y`; a direct challenge POST is rejected. Replay must authenticate rather than
merely navigate to a dashboard-shaped page. They cover password-only authentication, all eight reviewed
MFA kinds, main/popup/frame contexts, exact-name and unique-role locators,
zero/16/17 outputs, typed string/integer/number/Boolean/presence results,
noncanonical scalar rejection, stale and ambiguous targets, context
substitution, origin escape, secret-output rejection, disclosure-path
injection, and a fabricated parent/worker trace. The last two cases must fail
before profile staging. Every case starts a fresh loopback server, private
authoring root, browser context, Udon workdir, and Browserdriver lifecycle;
teardown is part of the result.

Run a bounded subset by repeating `--scenario`:

```bash
go run ./cmd/openudon browser-scenario-eval \
  --suite loopback \
  --stack current \
  --browserdriver-node-modules /absolute/read-only/browserdriver_node_modules \
  --scenario mfa-totp-scalars \
  --scenario popup-context \
  --require-ready \
  --out eval/runs/browser-scenario-loopback-local/report.json
```

Without `--require-ready`, a missing installed browser dependency is recorded
as `skipped` and an all-skipped report exits successfully as `not_run` for
structural inspection. `not_run` is never accepted as a pass; release
automation always requires readiness.

## Run The Realistic Journey Suite

The journey suite needs only the pinned Browserdriver Node Playwright Chromium;
it does not need a display or external network authority:

```bash
make browser-scenario-journey

# Run selected cases:
go run ./cmd/openudon browser-scenario-eval \
  --suite journey \
  --stack current \
  --scenario catalog-search-filter \
  --scenario record-update-approved \
  --require-ready \
  --out eval/runs/browser-scenario-journey-local/report.json
```

Its eight cases cover a search/filter form, pagination across two browser
operations in one named session, accessibility/JSON-LD/microdata/CSS reads,
an approved record update, rejection of the same update without operation
approval, an ambiguous mutation locator with no server write, four closed
parameter failures, and isolation between two complete executions. The local
application exercises `type_text`, radio and checkbox state, `select_option`,
click navigation waits, locator waits, typed outputs, exact mutation counts,
and final server state.

The current stack retains all eight cases and adds three required local cases:
Browser 1.8 path/query substitution with an exact signed 64-bit integer
supplied by a reviewed profile default,
Browser 1.9 literal-brace and text-sink substitution at the safe-integer
boundary, and a browser 1.5 then browser 1.9 action in one named v10 session.
The three modern cases use schema-checked local profiles and exact server
postconditions. The eight earlier cases keep their guided-authoring import.

Each of the eight earlier journey cases builds a deterministic
`browsertools.guided-authoring.v1` bundle
from normalized reviewed evidence, feeds it back through OpenUdon's strict
source importer, and materializes only the canonical browser 1.5 profile. The
private bundle, evidence, decisions, review, and draft spec never enter the UWS
package or report. OpenUdon then synthesizes ordered parameterized operations
in UWS 1.8, while Udon and Browserdriver replay them through protocol v3 in a
fresh headless Chromium lifecycle.

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

The hosted weekly workflow needs a repository Actions secret named
`GENELET_READ_TOKEN`. Use a fine-grained token with read-only **Contents**
access to `genelet/udon` and no other repository authority. The workflow reads
the exact Udon commit from the compatibility lock, checks out Browsertools and
Browserdriver anonymously, and checks out the private Udon revision with
credential persistence disabled. The repository-scoped `GITHUB_TOKEN` cannot
read that sibling private repository, and a missing credential fails before
Chromium or any public target is launched.

Public failures use typed live-result facts and strict
`udon.execution-report.v2` codes, never stderr text. Missing, malformed,
unknown, or unrelated failures are `unclassified`; that code can be recorded
but cannot satisfy a negative scenario. Other closed classes include
`target_unreachable`, `timeout`, `origin_policy_drift`, `shape_drift`, and
`contract_drift`. A
maintainer may add a manifest quarantine only for a documented upstream reason,
with fixed start/end dates no more than 14 days apart. Quarantine is visible in
the report and cannot silently become a pass.

## Report And Compatibility Contract

A failed loopback authoring run also writes an owner-only
`<report>.authoring-diagnostic.json` using
`openudon.browser-scenario-authoring-diagnostic.v2`. It contains only closed
scenario IDs, operation phases and failure codes, plus the SHA-256 of the
compact JSON scenario report. Its strict validator checks that binding and
the failed authoring phase; callers retaining evidence must also bind the
diagnostic bytes. Existing files and final symlinks are refused. No raw error,
page content, credential or token is included. Native qualification embeds the
same record in its private failure sidecar before temporary files are removed.
Successful report wire formats are unchanged, and failure diagnostics cannot
satisfy successful qualification.

Each v2 authoring entry requires a `failure` object with `worker_diagnostic`,
`stream_phase` and `stream_failure`. The last syntactically valid worker
diagnostic is reduced to an allowlisted producer code, `unknown` for any other
code, or `none` if absent. It remains separate from the controller's failure
code. Receive failures distinguish `eof`, `decode`, `size` and `read`; drain
failures distinguish `decode`, `size`, `read` and `trailing_message`. With no
observed stream failure, both stream fields are `none`. Clean EOF after a
terminal result is normal; child exit still must succeed before success is
published. Cancellation, deadlines and joined cleanup retain precedence.

The strict reader still accepts the original v1 shape. New fields, even null,
are rejected under v1; missing v2 details, unknown fields and unknown classes
are rejected under v2. These details are private metadata only: the browser
author event JSON and UI projection omit them, and native qualification retains
them only in its private failure sidecar. The author-session v2 protocol and
public report formats are unchanged. A retained `browser_failure` identifies a
producer category, not its underlying browser error or historical cause.

Historical loopback/public reports use `openudon.browser-scenario-eval.v1` and
historical journey reports use `openudon.browser-journey-eval.v1`. M86 current
local reports use `openudon.browser-scenario-eval.v2` and
`openudon.browser-journey-eval.v2`; their verifier uses frozen M86 lock bytes.
E21 current local reports use v3 versions, the frozen E21 lock, and complete
23-case loopback or 11-case journey inventories. New Browser 1.10 current
reports use v4 versions and its separate current lock; passing reports require
all 23 loopback cases or all 14 journey cases. The v4 journeys include
zero-, one-, and multiple-row `campaign_count` examples and retain only the
bounded output scalar. Filtered current runs remain diagnostics. Every report has an adjacent
`.sha256` sidecar and contains exact repository commits, public module
versions, closed phase/assertion/detail identifiers, counters, and explicit
safety booleans. It contains no target page content or subprocess output and
is safe to archive. Verify it independently:

```bash
go run ./cmd/openudon browser-scenario-eval \
  --verify eval/runs/browser-scenario-loopback-local/report.json
```

The embedded `openudon.browser-scenario-lock.v2` fixes Browsertools, Udon,
Browserdriver, UWS, Go, Node, Playwright, and Chromium compatibility. Execution
rejects a dirty or revision-mismatched OpenUdon or sibling checkout, a
mismatched OpenUdon module pin, or installed Playwright/Chromium versions that
differ from the lock. Generated `site/` output is explicitly excluded from the
dirty-root check and is neither removed nor release evidence. Scenario
manifests and reports strict-decode unknown or duplicate fields and apply
finite bounds before any browser or network authority is exercised.
The frozen E21 v3 lock fixes the UWS 1.11 stack and repaired Udon
`6d32d4967469c579d35adcf47eaddb76a225dbae`. Browser 1.10 v4 selects its
published UWS M05, Browsertools M32, Browserdriver M15 and Udon M43 commits.
Each generation requires its exact clean 14-repository Udon local replacement
closure, recorded in `current-qualification-build-inputs-v3.json` for v3 and
`current-qualification-build-inputs-v4.json` for v4. Its Node readiness probe launches pinned Chromium with
`chromiumSandbox: true`; readiness still does not replace an executed case.
When the clean Browserdriver checkout has no installed modules, pass
`--browserdriver-node-modules` with a separate directory outside that checkout.
Its `@types/node`, `playwright`, `playwright-core`, and `typescript` versions
must match the checkout's `package-lock.json`. Current-stack builds use these
modules read-only and place output in disposable directories; they do not
install dependencies or mutate supplied source worktrees.

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
credential export, and cross-execution browser-state reuse remain outside all
three scenario suites.
