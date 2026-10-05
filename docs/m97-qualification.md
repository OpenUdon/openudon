# M97 qualification and consumer handoff

## Exact source and build selection

Candidate source: `c55fb8eb208802955c8aac3e8ec6dc61b2d80bfc`.
The clean detached export excludes the owner's in-progress status changes,
all overlays and ambient `go.work`. Offline Go 1.26.6 builds use exact public
module pins; no private Udon import or download is involved. Source archive
SHA-256: `40a200a1051545b2043f799acce6dcfaacba3c943c871d36705bb98dc1d86580`.

| Artifact | SHA-256 |
|---|---|
| openudon | `38c0088b78a294da327971e31517bddd40d03fa2b376a2c6951d65ee4056b6da` |
| udon-runner | `7bc87ccfd3c5daefdfe7a2176e82356056ac06968cff098ef7db4fe6f0b7dbcb` |
| docs/fixtures/broker-execution-v1/manifest.json | `dd3357dc4a61e15f006e92f3d24fdbb72f88f9368336ee55375b08ed9ed0294f` |
| docs/fixtures/broker-handoff-v1/manifest.json | `903b0d5b885edc46df87e9158ee76fd7a3f422a62846dd4fed3d3f9584925539` |
| docs/fixtures/udon-http-broker-v1/manifest.json | `7289085b14f744504019ae6d607d351f74522ed3f930ecbbfd159200fd2ddb5f` |
| docs/schemas/openudon.broker-authority.v1.schema.json | `1fafe34b8de49b79b461879ab6e14da91f0b9f1616c89d7d75e90b5e5bf899e6` |
| docs/schemas/openudon.approval.v2.schema.json | `e269e2fe1d1dc3371819e205d0c8296821c80692d59f3f6ced424b3180b4216d` |
| docs/schemas/openudon.executor-run.v3.schema.json | `1db06ae5a484888c1e9aef9dac7a4bd31b3ff185b9c9b91ff489630bf164cbb1` |
| docs/schemas/openudon.run-evidence.v4.schema.json | `ce96682064c9d9bcb9e4e0b998b68d21e3ad379223ac598af1840fdaef68768c` |

The private build-input inventory binds 48 dependency modules, 2,784 compiled
source/embed inputs, the exact Go binary and module/sum bytes. The native
selection contains nineteen source entries (seventeen unique exports, because
Browsertools/UWS also occur in the fourteen-source M45 closure). Broker checks
retain the accepted M46 executor, source and closure below; the historical M45
browser lock is not rewritten. Both CLI binaries report `vcs.modified=false`.
An earlier preflight build saw a temporary owner-note edit in the export; it is
retained separately and excluded from this candidate/artifact selection.

## Observed verification

- `make fast`: passed on M97.3, with full Go/default tests and documentation memory checks.
- Focused portable corpus and broker tests: passed; focused trusted-runner races passed.
- Clean-export `make check`: passed (build, default tests, siblings and APItools boundary).
- Clean-export `go vet ./...`: passed.
- Clean-export exact M46 runtime qualification: seven of seven private-socket cases passed.
- Fresh affected registration capture smoke: passed, 74.059 seconds; development evidence only.
- Full `make qualify`: four offline gates passed and independently verified; three fresh native repeats remain in progress. No native/runtime acceptance claimed.

The initial full qualification stopped in its first offline gate when Go's
linker exhausted the existing `/tmp` tmpfs user quota. Its failed report and
private bounded diagnostic remain at the original context. Xvfb joined and
its socket/private authority were removed. The retry retains the same source
and assets, binding a private Go launcher that invokes the exact cached toolchain
with disk-backed temporary files and two build workers. It changes no stage,
validator or sandbox setting and consumes no cached qualification result.

## Producer provenance and downstream adoption

Udon M46 accepted source: `95c5850fd446e06ac6f79d943774db67e417c989`.
Completed owner publication: `71071537890599e98541abe8ca564490660dfd16`.
Executor SHA-256: `53bb9e8976f67c6a5880f07248a99cd195e6285ce7a5a68239793dd5dc6eb429`.
Closure SHA-256: `a286f21a5dcd4180b7ec19bf0630188ddacec26e0532198db14decd0c86b2069`.
Original fixture manifest: `7289085b14f744504019ae6d607d351f74522ed3f930ecbbfd159200fd2ddb5f`.

Portable OpenUdon fixtures are synthetic contract examples with independent
manifest hashes; they never replace actual runtime proof. See
[the handoff contract](broker-execution-handoff.md) for version/digest rules
and the original/consumer fixture links. The host consumer must enforce current
grants and credentials, destination/request policy and durable claims before
each I/O. Unknown writes are neither retried nor reclassified as safe failure.

Kinet M35/W14/M37 will reconcile the actual accepted and published OpenUdon
revisions and complete closure after owner qualification/review/publication.
Existing authoring/capture pin `c2f161d762bc9f2217bbf0c34b00cdef64b0f7d0`
and frozen browser evidence stay unchanged. W8M adoption, live provider/API
traffic, deployment and user/customer data are excluded.

## Private evidence location

`/var/tmp/kinet-stage9-p_9plhx7/m97-native/` contains the exact clean exports,
binaries, source archive, build-input inventory and original failed/smoke reports.
`attempt-2/`, `attempt-3/` and `attempt-4/` preserve separate retry contexts.
The second attempt still inherited Go's automatic child PATH prefix and hit the
`/tmp` quota; the third stopped on a disposable launcher typo before any
qualification stage. Every earlier display teardown passed. Attempt 4 uses
the exact clean compiled CLI through the owner Make entry point, keeping the
resource launcher on child PATH; its four offline gates passed with an
independent verifier, and native pass 1 stopped at registration_capture_handoff after ten successful stages. Focused diagnostic checks and the exact candidate component passed afterward; the original failure was not reproduced. Diagnostic overlays are excluded from native proof. Attempt 5 reruns the full unchanged qualification in a separate context.
`/var/tmp/kinet-stage9-p_9plhx7/m97-4-real-broker.log` records the clean-candidate
M46 runtime cases. Private transport capabilities never enter portable artifacts.

Additional M97.4 tests prove exact production approval admits the explicit
broker path, while public sandbox destinations and sandbox approval at the
production tier refuse. API-key header/query metadata reaches the private
executor binding without passing values. They are local injected-executor tests:
no DNS or public service is called. These test-only additions have separate
focused/default/race evidence; the candidate runtime, schema, fixture, module
and legacy browser bytes remain identical to the frozen native source.

Persisted pre-publication and ordinary closing reviews are not yet started.
Publication and final acceptance remain pending.


A task-level regression check found the async argv copy needed the same private-
config redaction as the main record. The assertion failed before fixing that
copy. Attempt 5 was interrupted before native stages; its outputs are excluded
from acceptance. The next clean source checkpoint includes this runtime fix and
the additional boundary tests. The c55fb8e artifacts above retain their original
pre-fix context and will not be relabeled as the corrected source.
