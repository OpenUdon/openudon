# A32 local source qualification

Go/test source is frozen at
`9e80ca8b955bd567cf5e3da17907f0f935ac2d0c`, following accepted combined
private-closure removal `f34e6017e87735fccd9893a185119621787359f8`.
The [final inventory](a32-final-consumer-inventory.json) preserves the separate
immutable [pre-removal baseline](a32-consumer-inventory.json).

| Observed local check | Result |
|---|---|
| Final inventory generation / exact regeneration | PASS, 1.36s / 1.00s |
| Baseline overwrite negative control | Refused as required; original SHA256 unchanged |
| Literal offline `make check` | PASS, 26.28s, exit0; build, `go test ./...`, `go vet ./...` |
| Kinet author source-fixture compile, coordinator | PASS, 20.6456s, exit0; browser-candidate tag, current A32 source bound by disposable modfile |
| Udon `pkg/execute` compile, coordinator | PASS, 15.3914s, exit0; unchanged source, no OpenUdon Go dependency |
| Public preservation | All 82 original public Go files and 159 schema/history/source fixtures retain exact baseline hashes |
| Installed strict MkDocs export, optional | FAILED, 2.58s, exit1; 13 cross-tree source/history links |

Make compiled the 19 current packages; ordinary test output retained 15 package
successes and four packages without tests. Individual RUN/skip counts are not
retained by ordinary Make output. The public API/AST/import guards are included.
The inventory records the unchanged 373-package public graph and empty owned
legacy package/import/command/Make-CI/HCL-input closure. Historical guide and
guard-fixture strings are reported separately.

Focused proof is reused at its actual scope: 84 tops/522 passing events for the
combined cut, six migrated public regressions/22 passing events, and A32.4's
API/AST guards/17 passing events. These are compile/unit/source checks, not
browser or private-runtime proof. Module pins, external codecs and historical
records remain unchanged.

The optional strict export's original 13 warnings comprise 12 links already
present at dispatch baseline `65504cc6` and one new A32 sibling-source link.
The new link was replaced with an explicit workspace source reference; its
hash remains in the inventory. Historical evidence links remain unchanged.
No strict rerun, validation weakening, nonstrict substitute or tooling repair
is claimed. Optional rendered-doc export remains deferred.

Logs, source preservation proof and link provenance were hash-verified into
the durable coordinator directory
`kinet.goal/stage12-coordinator/resume-20261013/a32-verification/owner-qualification/`.
No Go/test source changed after the full check.

The Kinet compile used actual retained worker source against current A32
source through a disposable fixture modfile. Original candidate module/sum pins
were unchanged; its binary was never executed. This is source-fixture
compatibility, not ordinary published adoption. Udon compilation similarly ran
no runtime. The coordinator retains exact plan/results/logs under
`kinet.goal/stage12-coordinator/resume-20261013/a32-verification/`, plan SHA256
`0bac1e7bdcc8131057a7410c7a3f11feec75233a8e8886d015f68ff195051fa7`.

A fresh empty bare Git store fetched both exact frozen OpenUdon commits
`01998d6f9d2d2bfd5e6bcbb2c5be9c0ec908b864` and
`c2f161d762bc9f2217bbf0c34b00cdef64b0f7d0` from the approved origin in
1.8773 seconds, exit0. No alternates, existing object store, tag or ref rewrite
was used. The coordinator retains `fresh-frozen-remote-result.json`.

Qualified source publication and fresh independent whole review remain pending
with the coordinator. This page grants no consumer adoption, deployment, runtime,
registration or private-input authority. Historical UNKNOWNs and no replay
remain; deferred Kinet physical/observer methods remain unqualified.
