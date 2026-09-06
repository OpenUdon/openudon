# Browser system qualification

`make browser-system-check` is the explicit local engineering gate. It runs
browser-free `offline` checks, verifies that report, then requires three fresh
complete `loopback` passes and verifies their aggregate report. Default `go test
./...` remains browser-free. No public target suite, package publication or
real registration authority is included.

```sh
openudon browser-system-eval --suite offline --out /tmp/browser-system-offline.json
openudon browser-system-eval --suite loopback --out /tmp/browser-system-loopback.json
openudon browser-system-eval --verify /tmp/browser-system-loopback.json
```

Use `--repo-root` for OpenUdon and `--udon-repo` for an exact disposable Udon
checkout with the auxiliary sibling checkouts required by
`internal/browserscenario/qualification-build-inputs.json`. The Make equivalent
is `OPENUDON_BROWSER_SYSTEM_UDON_REPO`. `OPENUDON_BROWSER_SYSTEM_OUT` selects
an output prefix outside both source workspaces. Root aliases are resolved
before enforcing that output boundary. Missing source revisions,
installed runtimes or sandbox prerequisites fail; the command never installs,
upgrades, downloads modules or substitutes a moving repository tip.

The baseline retains Playwright-Go v0.6201.0, Playwright 1.62.1 and Chromium
151.0.7922.34 on Linux. Browsertools and OpenUdon consume the same reviewed
Browsertools module. The UWS module implementation remains 9e676eaa469e while
its checkout closure is the reviewed documentation child cb5409586b7b. Udon's
local replacement lock binds the actual auxiliary source bytes independently
of its module requirement labels. A host requiring a Chromium setuid helper
must supply its administrator-owned `CHROME_DEVEL_SANDBOX`; Browsertools
validates the helper before accepting it. Qualification never disables the
Chromium sandbox to bypass readiness failures.

The loopback gate composes the existing scenario and journey corpus, real
BAP/BCP and BRP package qualifications, the sandbox-required browser UI suite,
Browserdriver's registration approval/uncertainty matrix, Udon browser protocol
and CLI contracts, and a real supervised command process. The command process
case covers stale revisions, observation, cancel, EOF, malformed input, retry
refusal and joined worker teardown against fresh synthetic fixtures. BRP qualification
now uses actual DOM controls and shipped JavaScript for every mutation from
worker start to package promotion; authenticated handler snapshots are
supplementary inspection. A separate tagged diagnostic runs that UI journey
without the Udon build prerequisites:

```sh
go test -tags=browser_system_qualification ./internal/icot/ui \
  -run '^TestBrowserSystemRealRegistrationUI$' -count=1 -timeout=6m
```

The report `openudon.browser-system-qualification.v2` binds each primary Git
revision and local source-tree digest, all auxiliary Udon build source trees
before and after each loopback stage, the compatibility/build-input locks,
runtime baseline, exact observed Go/Node versions, ordered component evidence and its SHA-256, scenario
inventory, and fixed failure stage. Local scenario components use aggregate-only source-bound validation; legacy
scenario publication files continue to require a clean OpenUdon checkout.
Local source deltas are engineering inputs;
they are not represented as published revisions. Component stdout and stderr,
page values, credentials and private producer files are not report fields.
Verification rejects unknown and duplicate JSON fields, altered evidence,
missing/reordered components, incomplete repeatability and skipped required
browser tests. A structurally valid report is integrity evidence, not a signed
attestation: independently compare its sources with the reviewed checkouts and
trust the process that ran qualification.

Command stages run under the existing process-group supervisor with bounded
execution and joined descendant teardown. EOF, cancellation and failures do
not authorize registration retry. Navigation/request allowlists are explicitly
reported as application enforcement, not network-wide containment. Synthetic
approvals and challenge values exist only in local fixture paths. Real-target
verification retains its separate human requirements.

# Supervised registration control

`icot control` accepts the workspace and package flags used by `icot ui`, and
requires `--no-open` with no listening port. It opens no controller browser and
uses no MCP configuration, result-text parser or agent-session binding. The UI
and this command call the same registration application methods and share
revision, one-attempt, worker, draft and containment semantics.

A supervisor owns private stdin/stdout pipes. Frames are bounded NDJSON:

```json
{"version":"openudon.registration-control.v1","operation":"snapshot"}
```

The closed operations are `snapshot`, `start`, `command`, `cancel`, and `close`.
`start` and `cancel` use the corresponding registration API-v4 request shape;
`command` uses its existing closed observe/navigate/draft/review/finish union.
Every mutation requires its exact current revisions. Review and finish retain
explicit confirmation. Output contains application revisions and reduced
registration state; that private channel must not be logged or used as a public
report. `close`, EOF, malformed input, cancellation and the nonrenewable
20-minute deadline cancel the worker and join its terminal state within the
separate teardown bound. A teardown failure remains a hard failure.

Browsertools registration is observation-only. The command cannot type into a
target, submit registration, resolve credentials or run arbitrary scripts.
Transactions, package preparation and trusted Udon execution retain their
existing interfaces and separate approval boundaries.

Use the source-bound aggregate report and the coordinating milestone ledger
for acceptance evidence. Source publication and operational adoption retain
separate review and authorization boundaries.

The supplied auxiliary baseline now pins Grand 5a3dae69ae44 and Hcllight
e2042c181d4a; Golet remains 38f8c62a8c31. These exact local replacement inputs
are tested separately from the Go module requirement labels. Progress output
contains only pass numbers and fixed stage identifiers.

Version 2 adds required supervised registration and authenticated-package
journeys to each of the three loopback passes (13 stages per pass). The verifier
continues to recognize version 1 using its original 11-stage inventory; a
legacy report does not qualify the application-control adapter.

For human browser challenges, an approved trusted run may explicitly opt into
`openudon run --interactive-browser`. This forwards private stdin through the
existing trusted runner to Udon. Default runs remain noninteractive, and neither
run configuration nor public evidence stores response values.
