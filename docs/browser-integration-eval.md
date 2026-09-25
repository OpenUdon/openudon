# Browser Integration Evaluation

`openudon browser-integration-eval` is the deterministic release-evidence
matrix for anonymous and authenticated browser authoring through trusted replay. It runs named,
provider-free checks in sibling OpenUdon, Browsertools, UWS, Udon, and
Browserdriver checkouts and writes one value-free report with an adjacent
SHA-256 sidecar.

M86 v2 reports use their frozen compatibility lock. E21 v3 reports use their
frozen current compatibility lock and 14-repository Udon build closure. New
current v4 matrices use the exact Browser 1.10 compatibility lock for UWS M05,
Browsertools M32, Browserdriver M15 and Udon M43, and validate the separate
clean 14-repository Udon build closure before running gates. OpenUdon and every
pinned sibling checkout must be clean. Historical v1 reports remain verified
against the unchanged scenario compatibility lock and original gate inventory.
Generated `site/` output is explicitly ignored without being removed or
treated as evidence. Browsertools
doctor inventory must report the lock's Playwright contract. The complementary
scenario evaluator launches the installed Node Playwright/Chromium pair once
and compares both actual versions with the same lock before replay.

```bash
make browser-integration-check

# Supply modules prepared outside the clean Browserdriver checkout:
make browser-integration-check \
  OPENUDON_BROWSERDRIVER_NODE_MODULES=/absolute/read-only/browserdriver_node_modules

# Equivalent direct commands:
go run ./cmd/openudon browser-integration-eval \
  --browserdriver-node-modules /absolute/read-only/browserdriver_node_modules \
  --out eval/runs/browser-integration-local/report.json
go run ./cmd/openudon browser-integration-eval \
  --verify eval/runs/browser-integration-local/report.json
```

The default run does not launch a browser, contact a target, read credential
values, execute a workflow, or retain subprocess stdout/stderr. It exercises
synthetic records and fake browser implementations, exercises iCoT's strict
live protocol/result adapters without launching a child browser, checks that
iCoT's engine has no Browsertools capture or Playwright implementation
dependency. The UI package has an explicit registration qualification adapter
that uses Playwright, but its dependency graph contains no Browsertools
capture implementation; ordinary matrix execution does not call that adapter.
The separately re-executed hidden worker remains Browsertools-owned. The
matrix runs Browserdriver's offline protocol tests and uses Browsertools doctor only to
observe pinned component availability without installation, browser launch, or
network access.

New current v4 runs accept `--browserdriver-node-modules` for the separately
supplied Browserdriver dependencies. Retained E21 v3 reports remain verifiable
with their frozen lock. The directory must be outside the clean
source checkout and its `@types/node`, `playwright`, `playwright-core`, and
`typescript` versions must match `package-lock.json`. The Browserdriver npm
test checks out the exact locked commit into a disposable clone, links those
modules read-only, and writes build output only there. No npm install or
supplied source worktree mutation is performed; missing or drifted dependency
versions reject the run before the matrix starts.
Current Udon Go gates likewise run in a disposable clone of the exact Udon
commit and all fourteen locked local replacement sources. The evaluator checks
the supplied source revisions and clean states after each gate and stops if
they drift.

## Required Matrix

| Gate | Evidence |
|---|---|
| OpenUdon authoring | API preference, anonymous handoff, strict author-session v2 orchestration, identical pre-publication validation for bundled and expert workers, disclosure-path rejection, human-only typed MFA/output review, exact new-origin approval, process-private trace/auth/output/context/origin attestation, minimal child environment, exact bounds authority, a real Browsertools-produced private result through validation/staging, and malformed/tampered/substituted rejection |
| OpenUdon package/handoff | Strict live and portability verification, private/tampered input rejection, value-free package review, authentication/capability separation, UWS 1.11 default, Browser 1.8/1.9 templates, Browser 1.10 count output, and v10/v11 trusted handoff |
| iCoT dependency boundary | The engine dependency graph contains no Browsertools capture, Playwright adapter, or Playwright-Go implementation package. The UI qualification adapter may link Playwright; the UI graph contains no Browsertools capture implementation. |
| OpenUdon repository boundary | Production source contains no private executor, desired-state parser, or removed apitools lifecycle imports |
| Browsertools producer | Observation-generation authority, human-selected MFA kind, bounded reviewed outputs, action-time exact-name/unique-role proof, complete context inventory, current goal proof, deterministic output, and offline doctor behavior |
| UWS contract | Immutable older compatibility plus UWS 1.11 typed conformance, root-scoped goto, Browser 1.8/1.9 template safety, context contracts, and scalar conversion |
| Udon consumer | Private source loading, runtime approvals, authentication, opaque sessions, v3 legacy replay, v10 Browser 1.8/1.9 handoff, v11 Browser 1.10 count output, UWS 1.11 bound execution, and post-conversion validation |
| Browserdriver runtime | Offline v2/v3 legacy NDJSON, v10 Browser 1.8/1.9 templates, v11 Browser 1.10 count output and integer safety, failure non-disclosure, exact-origin/context guards, credential lookup, and session isolation |
| Component inventory | Browsertools doctor reports pinned Chromium, Firefox, and WebKit readiness without installing or launching anything |

The current report contract is `openudon.browser-integration-eval.v4`. The v3
verifier remains bound to E21's frozen lock and v2 remains bound to M86's; v1
remains available for historical reports. Validation fixes
the gate order, repository names, command argv, assertions, authority claims,
counter totals, and closed value-free detail vocabulary. Passing Go gates must
contain every named test marker, not merely an overall package success. Reports
also require the producer-to-consumer and producer-to-replay test names; a
hand-built compatible fixture does not establish either seam. Browserdriver's
npm gate similarly requires its named replay and cached-context freshness
tests rather than inferring coverage from a passing-test count. Reports
record the commit and dirty-worktree bit for every participating repository,
are written atomically under ignored `eval/runs/`, never include repository
paths or captured command output, and can be verified only with their matching
digest sidecar. Verification rejects a structurally valid report whose matrix
status is `fail` when used as a release gate.

The Go module pins name the exact Browsertools and UWS feature commits used by
the seam tests. Compatibility validation checks both OpenUdon's Browsertools/UWS
requirements and Browsertools' own UWS requirement against the same lock.
During a coordinated pre-publication review, standalone tests may use
process-local Git URL mappings to clean local clones of those exact commits.
Release evidence must use ordinary module resolution after the commits are
published; an unreachable pseudo-version or a committed `replace` is not a
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

M86 qualification requests both flags together and requires all three opt-in
gates to pass, producing 19 passed, zero failed and zero skipped gates. A
doctor inventory result alone is not evidence of a browser launch.

The generated report is release evidence, not runtime authority. Keep any
private Browsertools cache, guided evidence, assisted-authentication bundle,
raw/rich capture, cookie, storage state, or live session outside OpenUdon
packages and outside this report.
