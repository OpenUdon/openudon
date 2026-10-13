# P12 public and pure native source compatibility

The P12.3 external fixture passes all ten cases against frozen public source
`9352d394bfd2dabc4a627ccf4ca827d06526c737` and accepted native Udon
`5bbf2924d85b9f13b89bb8c000cb1b12e6c56491`. This is pre-publication producer
source compatibility evidence. Ordinary published-module verification, Kinet
worker adoption and physical browser qualification remain separate gates.

The [complete qualification record](p12-compatibility.json) retains the external
test, module/sum files, synthetic profiles, exact authored workflow bytes and
SHA256 identities. Its module file is seeded from the frozen M52 browser-candidate
manifest. Its only local replacement is the P12 public SDK source. The existing
Docker-to-Moby version replacement is retained. Private imports exist only in
the isolated external test; no private runtime enters OpenUdon's public graph.

## Passing matrix

| Case | Result |
|---|---|
| Authentication 1.0 and 1.1, each with authored 0.25, 30 and 600 seconds, followed by read and write actions | Build/Assess/Verify, derive/check plan, canonical finalized approval, CheckBrowserApproval and derive/check Authority pass; actual native Compile/CheckSupported pass on the same verified workflow/data/source snapshot. |
| Each profile rebuilt from 30 to 31 seconds | Workflow, package, input, handoff, raw constraints and plan identities change; source and selected-profile provenance stay identical. Old or rehashed constraints, old plan, Config, receipt and Authority refuse. Timeout-only tampering under the old proof refuses Verify. |
| Each profile without a timeout | Public Build/Assess/Verify and read-only history remain valid. Actual native Compile returns `*execute.Refusal{Code: InvalidPlan}` before reading sources; the public admission adapter propagates that refusal. Removing timing from an exposed timed native document also makes CheckSupported refuse before source access. |

Every native host effect method shares a fail-fast counter. All eight BrowserHost
methods and both BrowserInteraction methods remain at zero. Compile reads exactly
two cloned in-memory source snapshots; CheckSupported reads none. The fixture calls
only Compile and CheckSupported. Reports, outputs, transport, credentials, Guard,
HTTP broker, durable session and optional host clock remain absent.

The runtime admission callback compares its workflow, data and complete source
inventory with the VerifiedPackage snapshot. It checks preparation metadata before
the public plan exists, using explicit non-authorizing preparation hashes. A
separate native check then binds the actual derived plan and exact finalized
receipt hashes, preserving the original synthetic admitted deadline. Synthetic
worker binaries, closure facts, containment references and the inert absolute
executable path establish no compiled worker, launch, session or host grant.

## Verification and diagnosis

Cached Go 1.26.6 with workspace/proxy/VCS/checksum-network/authentication helpers
and toolchain acquisition disabled resolves the 516-package direct-import graph
in 0.046 seconds. The first full test invocation compiled but refused native
1.10 fixture outputs; a narrowed diagnostic confirmed refusal after two source
reads. Accepted native 1.10 requires nonempty CSS match-count outputs on every
action. Only the fresh synthetic profiles were corrected to that contract.
The subsequent full focused invocation passes all ten cases in 4.452 seconds,
with zero effect callbacks. The qualification record retains both failed
invocations and the correction; no native semantics or historical source is
changed. A later test-only `go list .` is explicitly not test-graph qualification.

Existing P12.2 public tests separately retain invalid timing, nonfinite predicates,
numeric lexemes, extensions, named-output semantics, non-browser controls and
old untimed/golden behavior. This record claims no owner closing gates, milestone
review, acceptance, publication or consumer adoption.

## Reproduce outside the public repository

Materialize each `fixture_files` entry from the JSON record into a separate
external directory and verify its `sha256` before running it. Preserve the exact
recorded module/sum files and ordinary cached native/C10/M33 versions. Adjust
only the SDK local replacement if the independently verified public source is
mounted elsewhere; that remains a source fixture. Do not place the private
test inside OpenUdon or replace the native module with a sibling checkout.

Use a fresh empty process environment and the record's explicit `environment`
mapping, retaining the existing module/build caches. From the external fixture,
run these bounded commands:

```sh
timeout 120 go list -deps github.com/OpenUdon/openudon/packagev3 github.com/OpenUdon/openudon/approval github.com/genelet/udon/pkg/execute
timeout 120 go test -run TestP12 -count=1 -timeout=90s -v .
```

The successful run prints workflow, package, inputs, handoff, raw constraints,
plan, receipt, Config and source digests. The source evidence is dated
2026-10-13 with its unchanged native `P30D` freshness period; a later expired
fixture must produce the native refusal rather than fabricate continued support.
