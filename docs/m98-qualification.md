# M98 public trust qualification

Qualified implementation source is
`08a3839f357ec40c7e50c8668e4bd7c8d86bb55a`.
[Build inventory](m98-qualified-build.json) binds the source archive, Go 1.26.6,
module/sum bytes, API surface, final check log and both CLI artifacts.
[Ordinary module archives](m98-module-archives.json) records all 89 selected
dependency versions, Go module/go.mod sums and exact archive SHA-256 values
in the 90-module owner graph. No directory replacement or version upgrade is
used by the qualified owner build. Seven previously uncached already-selected
archives were acquired under the confirmed build-closure authority.

The clean ordinary clone passed `GOWORK=off GOPROXY=off make check` and vet
with cached Go 1.26.6 and `GOTOOLCHAIN=local`. Public standalone tests, import
and API surface guards, trust/approval/broker/report wire fixtures and focused
public/CLI/runner race tests passed. Kinet's unchanged `make check` passed.
The independent standalone SDK consumer passed against the exact source with
a temporary local owner bootstrap; this is preliminary consumer evidence.
[Publication](m98-publication.md) records the subsequent successful ordinary no-directory module proof.

| Artifact | SHA-256 |
|---|---|
| Source tar | `2e301ed7de7f69aa669ac6a970b832ad6260fb337173c6455a0be286cbe4fed7` |
| openudon, reproduced identically | `6e185ecfe7d38efc6ca8cb0c01938aa50a422e9b1e491ea4d2c3e7ceb10fdb95` |
| udon-runner | `556258a798cf54e90d0042c9fa06a461960ed86696dccf56675c35f9c1ccb256` |
| Existing API surface fixture | `8ac50a0eb84ba7a28ab2fda375bfc536b0df568272017e3f0674b9ca315b1e7e` |

Both CLI build records carry the full qualified source and
`vcs.modified=false`. Early Git-worktree builds omitted those stamps, so
qualification uses an ordinary clean clone. An early output followed a scratch
symlink into the repository; its generated untracked developer binary was
saved, hash-checked and removed. Those earlier builds are diagnostic only.
All final artifacts use dedicated output files. Source-6b355b6 preliminary
results were superseded by review fixes; none establishes final acceptance.

Pre-publication review iteration 1 found two P2 defects: trim-permissive
authority hashes and incomplete broker raw-wire verification. Canonical
field checks and the unchanged embedded schemas with a closed loader fix
both. Full tests, races and clean-source qualification pass after those fixes.
Pre-publication iteration 2 and closing iteration 3 passed; exact publication and acceptance are recorded in the [retired owner status](../tabilet/docs/history/status-M98.md).

The public contract is [public-trust-api.md](public-trust-api.md). Public
packages expose neutral metadata and explicit-byte verification, without
OpenUdon internal/private executor imports or stable synthesis-coupled v2
construction. Browser execution/verification retain the separately pinned
legacy CLI path. Published schema/fixture bytes and all frozen pins are
unchanged. Existing offline browser adapter regressions passed; no browser
runtime/capture implementation or UI changed, so no fresh native browser run
was selected. Source publication grants no deployment or live execution.

Disposable evidence root: `/var/tmp/om98-qualification-20261007/`.
