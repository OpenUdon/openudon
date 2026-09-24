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

| Item | State | Notes |
|---|---|---|
| M84.1 | `[+]` | Pinned published UWS `e9b6181be0ab`; new synthesis emits 1.11.0, while loaded documents keep their declared versions. Updated generated-scenario expectations and three expired synthetic registration fixtures. Workspace and pinned `go test ./...`, `go vet ./...`, local CLI checks, example validation, and `git diff --check` passed. |
| M84.2 | `[ ]` | After owner contracts are reviewed, accept Browser 1.8/1.9 in source/review/package/handoff paths, pin Browsertools, reject unsupported runtime combinations, and cover safe parameter/template flow with synthetic tests. |
| M84.3 | `[ ]` | Reconcile public and memory-bank docs, run workspace and pinned builds/tests/vet plus one synthetic browser integration check, review the whole milestone, and record final compatibility evidence. |

**Verification and review.** Run `go test ./...`, `go vet ./...`,
`go run ./cmd/openudon check`, `go run ./cmd/openudon check-apitools-boundary`,
`go run ./cmd/openudon check-doc-memory`, relevant package/example validation,
and `git diff --check`. Compare `GOWORK=off` with workspace mode after pins are
published. Review the milestone against acceptance before closure; row
completion is one scoped commit each under local policy.

**Review gate.** Not started.
