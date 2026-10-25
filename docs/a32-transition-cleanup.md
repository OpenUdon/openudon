# A32 removal inventory and handoff

The immutable [A32.1 pre-removal inventory](a32-consumer-inventory.json) records
76 owner packages, 373 public transitive packages, 565 Go files, six retained
A31 targets, five production HCL input calls in four owners, and one legacy HCL
round-trip test. Its SHA256 is
`8851131ff8935f8ca82f35cf0f7fcaaa19c55e631e4c45c1766fdbf4d791f382`.
The [A31 inventory](a31-consumer-inventory.json) and its original handoff remain
historical evidence. A32.1 is reproducible with its original generator and
recorded inputs at commit `cd57f0c9c4c42e9009dd4de10144967b154c3dfa`;
its production/module source identity is `7356d428a406100105d843bbf526ae1b5867d47d`.
The current generator preserves that baseline and writes a separate
[final inventory](a32-final-consumer-inventory.json).

Regenerate or check the final inventory with Python's standard library and
existing pinned Go 1.26.6/cache inputs:

```sh
export PATH=/home/peter/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.6.linux-amd64/bin:$PATH
export GOMODCACHE=/home/peter/Workspace/openudon.goal/verification-P10-4m_d2i4b/gomodcache-restored
python3 tools/consumer-inventory/generate.py --workspace /home/peter/Workspace --w8m /home/peter/Ready/w8m-browser-operations
python3 tools/consumer-inventory/generate.py --workspace /home/peter/Workspace --w8m /home/peter/Ready/w8m-browser-operations --check
```

The generator enforces offline, standalone, local-toolchain and read-only module
selection. It refuses to overwrite the pre-removal baseline, checks every
retained public source/schema/history fixture hash against that baseline, and
records the last Go/module source revision. Tracked Go files are inventoried;
new product files must be tracked before regeneration. Default `go list` graph
edges and all-file lexical import edges remain distinct. Lexical tagged-file
coverage does not claim tagged code executed.

Accepted combined source `f34e6017e87735fccd9893a185119621787359f8`
removed the private authoring, intent/synthesis, browser/capture harness,
runner/evaluation/helper and CLI closure. Three public API guards migrated
unchanged to `tests/publicapi`; six unique retained policy/report/authority
success and refusal regressions migrated to public API consumers under `tests`.
The [retirement guide](legacy-cli-retirement.md) records current owners.
Source `9e80ca8b955bd567cf5e3da17907f0f935ac2d0c` adds a standard-library
AST guard covering all owned Go source, including inactive build tags, and
extends public dependency closure checks to all 13 roots.

The final inventory distinguishes actual owned package/import/command/Make/CI
closure from historical command examples retained in guides and frozen
artifacts. Guard test strings intentionally name refused HCL entrypoints;
these are recorded separately from owned input references. The semantic AST
regression rejects aliased/dot/function-value HCL input references, while
history bytes and external codec output dependencies remain legal. No claim
that every legacy string vanished is made.

P10 construction/verification/transaction/approval/evidence, P11 named-output
ownership/deterministic assessment/stale-proof refusal, P12 authored bounded
authentication and readable untimed history, P13 saved-metadata expiry/observed
fallback rights, all v2/v3 history readers and public manifests remain unchanged.
All 82 original public Go files and 159 schema/history/source fixture files
retain their exact A32.1 hashes. External Horizon/HCL codec dependencies and
`go.mod`/`go.sum` pins remain unchanged.

The Kinet M53 source handoff at
`/home/peter/Workspace/kinet/docs/m53-stage12-handoff.md` binds retirement
of the old OpenUdon CLI/image/internal producer path for accepted native and
retained default/Phase A/media paths. Public libraries remain live: browser
workers use P13, independent default workers retain801b, and private Udon/HCL
remains. Extra physical/observer/forced-crash methods stay unqualified; historical
UNKNOWNs, no replay, registration denial and disabled private input persist.
These deferred methods add no A32 prerequisite or runtime authority.

Local Git inspection resolves frozen W8M `b23fd637` and its OpenUdon
system-history `01998d6f` and candidate `c2f161d7` objects. Lock bytes and their
candidate qualification state are preserved. The exact recorded Ramen snapshot
has no OpenUdon module requirement. Final source checks are recorded in
[A32 qualification](a32-qualification.md). Retained consumer builds, exact
remote fetchability/source publication and fresh whole review remain
coordinator-owned acceptance gates; this inventory supplies no completion or
runtime claim for those gates.
