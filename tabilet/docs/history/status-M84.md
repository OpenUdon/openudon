# Retired milestone M84 - M84 — UWS 1.11 and Browser 1.8/1.9 adoption

**Milestone.** M84
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M84.md
**Source status SHA-256.** f946cfe60e8bc333aba35ab95ff8debe98b2f750c1682dfc1a455c2a02354677
**Source milestone snapshot.** tabilet/docs/history/milestone-before-legacy-retirement.md.txt
**Snapshot SHA-256.** 26884eeda9af4ded30e6d545a33dca84c401d2320c22355fe4fb0336d5dcac71
**Evidence.** 71a4f78afbcf2180fc478ffa89c53544c9160648
**Worktree.** includes uncommitted changes
**Review.** not established
**Review iterations.** not recorded
**Verification.** Original status bytes and full earlier milestone bytes preserved by SHA-256; no fresh acceptance claim.
**Consolidated into.** Current milestone dashboard and maintained memory-bank guidance; full earlier text remains in the frozen snapshot.

## Status record

~~~~~~~~~~~~~~~~~~~~markdown
# M84 — UWS 1.11 and Browser 1.8/1.9 adoption

**Goal.** Newly generated OpenUdon workflows declare UWS 1.11.0 by default,
and reviewed Browser 1.8/1.9 profiles can pass authoring, packaging and the
trusted runtime handoff with compatible Browsertools, Browserdriver and Udon
implementations.

**Boundary.** Existing packaged documents keep their declared versions;
historical sources, browser approvals and private registration authority do
not change. Do not treat a workspace `go.work` build as proof that pinned builds
resolve the published module revisions. No live browser or target operation is
authorized by this milestone.

**Provenance.** The user approved full 1.11 adoption, 1.11 default output and
Browserdriver inclusion after reviewing `uws-downstreams.md` (2026-09-24).
Revalidated at OpenUdon HEAD `57e516fd06abc823a1eee8eb909c6b3bfe1c9144` and
published UWS HEAD `e9b6181be0abb7f683fdb624d4dba282a59991d1`, with clean
worktrees. The review found no UWS-caused current breakage. OpenUdon's three
registration fixture tests fail after their 2026-09-24 expiry; that is separate
from UWS adoption.

**Dependencies.** M84.1 requires the published UWS revision. M84.2 consumes
reviewed Browsertools Browser 1.8/1.9 support and Udon/Browserdriver's private
action protocol. M84.3 requires both rows. Ramen is outside this milestone.
Published reviewed revisions are Browsertools `9333a9f25dbb17551998a429e123e7a9ba976648`
(M30/M31), Browserdriver `8c13b70d30a500e65e90a95a203493301b8b21a5`
(M14/M14.2), and Udon `080b8282e2b8f7ca7a9994b6d9f0e3d2891d853f`
(M42). OpenUdon resolves Browsertools through its published pseudo-version.

| Item | State | Notes |
|---|---|---|
| M84.1 | `[+]` | Pinned published UWS `e9b6181be0ab`; new synthesis emits 1.11.0, while loaded documents keep their declared versions. Updated generated-scenario expectations and three expired synthetic registration fixtures. Workspace and pinned `go test ./...`, `go vet ./...`, local CLI checks, example validation, and `git diff --check` passed. |
| M84.2 | `[+]` | Browser 1.8/1.9 source/review/transaction paths and active-source v10 runtime selection pass synthetic template, iCoT candidate, mixed-profile, and local/Docker runner handoff tests. Browsertools is pinned to published `9333a9f25dbb`; workspace and `GOWORK=off` `go test ./...`, `go vet ./...`, standalone build, module verification, CLI checks, example validation and diff check pass. Row review found no open P1/P2. |
| M84.3 | `[+]` | Public and memory-bank docs describe UWS 1.11 output, Browser 1.8/1.9, v10 handoff and the historical matrix lock. Final workspace and `GOWORK=off` full tests/vet, standalone build/module verification, focused synthetic browser handoff tests, CLI checks, example validation and diff check pass. Udon's local Browserdriver loopback smoke exercised Browser 1.9 and a legacy action in one authenticated session. Whole-milestone review iteration 1 found no open P1/P2. |

**Verification and review.** Run `go test ./...`, `go vet ./...`,
`go run ./cmd/openudon check`, `go run ./cmd/openudon check-apitools-boundary`,
`go run ./cmd/openudon check-doc-memory`, relevant package/example validation,
and `git diff --check`. Compare `GOWORK=off` with workspace mode after pins are
published. Review the milestone against acceptance before closure; row
completion is one scoped commit each under local policy.

**Review gate.** Complete at iteration 1. Reviewed M84.1 and M84.2 changes
against the UWS 1.11, Browser 1.8/1.9, private v10 handoff and historical
compatibility boundaries. No open P1/P2 finding remains. The pinned
Browsertools module resolves from the published revision; published
Browserdriver and Udon revisions match the locally tested owners.

**Historical matrix boundary.** `make browser-integration-check` was tried
against the M84 local stack and correctly stopped before running gates because
the W13 browser-scenario compatibility lock fixes older Browserdriver,
Browsertools, Udon and UWS revisions. That lock and its qualification history
remain unchanged. M84 uses new focused OpenUdon handoff tests and Udon's local
Browserdriver loopback smoke for adoption evidence.
~~~~~~~~~~~~~~~~~~~~
