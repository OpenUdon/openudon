# A32 current removal inventory

[Current inventory](a32-consumer-inventory.json) refreshes the immutable
[A31 inventory](a31-consumer-inventory.json) at the source baseline named in the
JSON. It records all 76 owner packages, default production/transitive/test
imports, all tracked Go import declarations including tagged files, command
cases, build/doc references, and exact source/fixture hashes. This is source
inventory; it executes no workflow, browser, native worker or consumer build.
The six A31 targets still have the same production/transitive/test consumers.
P10 added the three public browser packages since A31 (73 → 76 owner packages;
353 → 373 public transitive packages).

Regenerate or check with Python's standard library and existing Go 1.26.6:

```sh
export PATH=/home/peter/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.6.linux-amd64/bin:$PATH
export GOMODCACHE=/home/peter/Workspace/openudon.goal/verification-P10-4m_d2i4b/gomodcache-restored
python3 tools/consumer-inventory/generate.py --workspace /home/peter/Workspace --w8m /home/peter/Ready/w8m-browser-operations
python3 tools/consumer-inventory/generate.py --workspace /home/peter/Workspace --w8m /home/peter/Ready/w8m-browser-operations --check
```

The generator enforces offline, standalone, local-toolchain, read-only module
selection. It records the last Go/module source revision, so ordinary inventory
or ledger commits do not invalidate an unchanged source snapshot. Exact Go-file
hashes identify the inspected working source. Tracked files only are inventoried;
new product files must be tracked before final regeneration. Platform/default
`go list` edges and all-file lexical import edges are deliberately separate.
The lexical scan covers tagged imports; it does not claim tagged code compiled.

The declared removal set covers A32.2 browser authoring/harness packages and
A32.3 retained authoring/runners/evaluation packages, including nested browser
transaction engine/presentation and `cmd/udon-runner`. Its only external direct
production consumer is `cmd/openudon`. All current command cases and 81
build/doc reference files are in the JSON. Retire the legacy commands; any
retained CLI must use public packages only. The `other_internal_packages` list
makes the remaining helper/alias/test closure explicit: remove unused private
helpers with their owners; migrate needed public API/wire regression tests from
`internal/publicapi` before retiring it. Do not silently move a legacy helper
into the supported public API. Later rows decide each concrete deletion.

A32.4 must clear five production UWS HCL input calls in four owners:
`synthesize` (two), `udonrunner`, `internal/udonreport`, and `uwsexec`, plus the
legacy HCL round-trip test. Removing the six A31 targets alone is insufficient.
The inventory also records intent/workflow HCL documentation. Retained public
Horizon/HCL codec dependencies are distinct from these OpenUdon HCL input
adapters; their presence is not permission to remove public source contracts.

Preserve every public root and embedded schema, the source hashes and historical
wire fixtures in the JSON. In particular retain P10 construction/verification/
transaction/approval/evidence, P11 named-output ownership and deterministic
assessment plus stale-proof refusal, P12 bounded authentication construction and
plan predicates with untimed history/native refusal, and P13 requested authority
versus observed fresh-fallback rights and saved-metadata expiry regressions.
Keep `packagev3` v2/v3 history readers and the API/wire manifests. Tests tied to
retired private behavior may retire; meaningful success/refusal coverage for
retained public behavior must survive through these public successors.

The [Kinet M53 handoff](../../kinet/docs/m53-stage12-handoff.md) is hash-bound in
the inventory. It establishes retirement of OpenUdon CLI/image/internal producer
selection for accepted native and retained default/Phase A/media paths. Public
libraries remain live (P13 browser workers and independent default801b workers),
and private Udon/HCL remains. Extra physical/observer/forced-crash proof stays
unqualified; historical UNKNOWNs, no replay, registration denial and disabled
private input persist. Those deferred methods add no A32 prerequisite.

Read-only local Git inspection resolves the frozen W8M baseline `b23fd637` and
its original OpenUdon system-history `01998d6f` and candidate `c2f161d7` pins.
Both lock contents and hashes are retained without changing the candidate's
unqualified state. The exact inspected Ramen snapshot has no OpenUdon module
requirement; its conversion/private-runtime boundary remains Ramen-owned.
Local object availability is established here. Remote fetchability, retained
consumer builds, final empty closure/public-boundary checks and source publication
remain A32.5 owner gates. No sibling pin, history, tag or checkout is changed.
