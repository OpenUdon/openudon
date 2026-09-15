# Browser system qualification

Routine development uses `make fast` and one affected authorized `make smoke`.
`make qualify` (also `make browser-system-check`) is the complete integration gate. It runs
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
Browsertools module at `ec0b9e9d6ca1`. UWS implementation and checkout are
`9ff877ebce55`; Udon is `5ef6af99430c` and Browserdriver is `9d13e8b35394`.
The BRP package qualification now exercises actual typed iCoT authoring,
transaction v3, UWS call 1.1, the separate private form's Start/Apply/approval,
and exact protocol-v5 replay. It verifies one POST and scans package/run files
for the synthetic private values and accepted snapshot digest. Udon's
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
execution and joined descendant teardown. On Linux, fast owned-child discovery
runs independently of whole-host scans; teardown joins both monitors while
retaining recorded PID/start-time ownership. EOF, cancellation and failures do
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
contains pass numbers and fixed stage identifiers. On failure it also prints
the location of an owner-only `<report>.diagnostic.json` sidecar. That private
file records a fixed failure reason and up to the last 1 MiB of each captured
child stdout/stderr stream, with a truncation marker. It never enters the reduced
report, is not qualification evidence, and must remain outside Git. Existing
diagnostics and final symlinks are never overwritten. A supervising composer
must preserve this sidecar before deleting its temporary component directory.
In-process loopback and journey failures retain their returned scenario report
and cause in those same bounded private fields, including failed-case and phase
details that ordinary progress intentionally omits.

Version 2 adds required supervised registration and authenticated-package
journeys to each of the three loopback passes (13 stages per pass). The verifier
continues to recognize version 1 using its original 11-stage inventory; a
legacy report does not qualify the application-control adapter.

For human browser challenges, an approved trusted run may explicitly opt into
`openudon run --interactive-browser`. This forwards private stdin through the
existing trusted runner to Udon. Default runs remain noninteractive, and neither
run configuration nor public evidence stores response values.

## Focused development

The verification candidate advances `registration_ui_handoff` to BRP/call 1.2,
author-session/result v4, transaction v4 and driver v6. It reviews a synthetic
Turnstile widget through the actual iCoT UI, packages it, and submits once through
the private input form and trusted executor. The native synthetic driver stage
also exercises all three providers and both activation modes. Existing native
report versions and historical proofs remain verifiable. Provider-network test
keys and production acceptance remain separate.

Before publication, the UI-only development test can consume an explicitly
prepared Browsertools binary through `OPENUDON_TEST_BROWSERTOOLS_EXECUTABLE`
and run `TestBrowserSystemVerificationRegistrationUI` with the
`browser_system_qualification` build tag. This neither qualifies nor adopts a
runtime; complete acceptance still builds frozen published sources.

```sh
make fast
make smoke OPENUDON_BROWSER_SYSTEM_UDON_REPO=/absolute/prepared/udon \
  OPENUDON_SMOKE_OUT=/tmp/unique-smoke.json
# Explicit reuse of a matching success younger than 24 hours:
make smoke OPENUDON_BROWSER_SYSTEM_UDON_REPO=/absolute/prepared/udon \
  OPENUDON_SMOKE_OUT=/tmp/unique-reuse.json OPENUDON_SMOKE_REUSE=true
```

`fast` runs unit tests with normal Go result caching and document checks. Smoke
runs one closed stage; `OPENUDON_SMOKE_STAGE` defaults to
`registration_ui_handoff`, the typed UI/BRP/UWS/private-form/trusted-runtime
flow. Other native loopback stage IDs can be selected explicitly. Unit/helper
or documentation changes normally need browser-free checks only. Use the full
fresh gate after affected smoke passes when qualifying a frozen candidate or
adopting runtime bytes.

The thin CLI is `openudon browser-system-dev --mode fast|smoke --stage ID
--repo-root ROOT --udon-repo UDON --out FILE [--cache DIRECTORY] [--reuse]`.
Output must be new and outside both source workspaces. No browser state is
reused. Only a smoke context can activate the private immutable build cache;
qualification remains fresh. `OPENUDON_SMOKE_CACHE=` disables custom caching.
The cache copies explicit Browserdriver/Udon/Browsertools build outputs into
fresh disposable runtime paths. Other child test stages retain their existing
Go cache and disposable builds.

The input key covers dirty source/fixture bytes, actual Go dependencies and
toolchain files, installed Node/npm/JS modules, checker executable, Chromium
runtime distributions, sandbox bytes and environment. Keys contain hashes;
reports contain no environment values or command arguments. Source drift during
a fresh stage fails it. Corrupt cache entries fail closed. An expired matching
result can be replaced by a fresh run without `--reuse`; corrupt artifacts need
cache removal or a run with custom caching disabled. Missing or changed keys
run fresh. Caches are owner-private, local integrity aids, not signed evidence.

`openudon.browser-development.v1` reports always say
`qualifies_runtime:false`. Explicit reused successes retain the original
execution ID/time/duration/proof and set `reused:true`; they never count as
independent repetitions. The qualification verifier rejects development reports.

Both native qualification and development write private `FILE.timing.jsonl`
sidecars (`openudon.browser-check-timing.v1`). Closed labels measure source
hashing, subprocesses, stage duration and supported builds/transaction teardown.
Timing is diagnostic, separate from the unchanged native evidence schema.

Custom build/result reuse is supported for the in-process
`registration_ui_handoff` and `bap_bcp_transaction` stages. Other selected stages
run fresh using their existing build behavior; `--reuse` for those stages is
refused. Their child environments are not treated as interchangeable cached
build inputs. The fingerprint also includes the installed Playwright-Go driver
and host package/kernel identity, separately from Node's Playwright modules.

## Registration foreground and deadline integration

Browserdriver M12 requests foreground presentation for the registration page
at launch and input/human/submit checkpoints. Udon M40 uses the earliest known
driver, broker and enclosing workflow deadline for the private form countdown.
The typed UI-to-runtime qualification requires that countdown at each actual
private checkpoint. Reload preserves its deadline; expired form actions are
rejected independently of the UI. Initial preparation remains untimed.

This integration does not change public BRP/UWS, private input identity,
registration authority, retry refusal or the W8M consumed-attempt history.
Use one fresh affected smoke while developing, then full acceptance v2 for
the frozen candidate before adopting new runtime bytes.

## Browser-free qualification input identity

`openudon browser-system-input --repo-root /absolute/immutable/openudon
--udon-repo /absolute/prepared/udon` emits only version
`openudon.browser-qualification-input.v1` and a SHA-256. It launches no browser,
runs no qualification stage and grants no execution or cache authority. Errors
are closed (`qualification_input`); paths and environment values are not output.

The inventory binds all nineteen source closures and exact root locations,
including nonignored edits; the running checker; Go, Node, npm and Git; Go dependency
files from the union of default, iCoT UI and qualification-tag builds; installed npm/Playwright,
Chromium and Go-driver trees; sandbox bytes; effective environment; host package,
kernel/boot identity, namespace policy and display-authority contents. Sources
and host state are checked again before returning. The qualification child
requires umask 0022, matching existing permission-refusal fixtures. Rebuilding
from a changed path or changing an input requires a new identity.

A downstream consumer can use this identity in its own explicitly versioned
reuse policy. W8M acceptance v3 owns its 24-hour native-cache admission, original
execution provenance, maintained supervisor binding and three fresh consumer
journeys. Native `browser-system-eval` remains a fresh three-repeat gate; its
report version and verifier semantics are unchanged. Development-cache results
remain ineligible for native qualification and cannot seed W8M's cache.
