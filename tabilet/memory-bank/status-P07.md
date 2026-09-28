# Status P07 — Browser 1.10 trusted package dispatch compatibility

**State:** Active. P07.1 is complete; P07.2 is in progress. Local branch
`P07`; no push or publication.

**Goal.** Repair OpenUdon's trusted-runner rank-10 dispatch so the existing
Browser 1.10 count profile passes the complete package preparation and
trusted-dry-run lifecycle while preserving Browser 1.8/1.9 behavior.

**Review source and finding.** F01, W8M W22 bounded review iteration 4,
`tabilet/memory-bank/status-W22.md` in the W8M repository, W22.5 offline
package-preflight investigation. Source priority P2; local severity P2;
disposition confirmed; owner OpenUdon P07. The review baseline commit was not
supplied. The W8M status-file evidence was uncommitted in the inspected W8M
main worktree. OpenUdon revalidation used clean commit
`55b24d29279c8efe67ae931f9d73f094f929efab`: its trusted runner accepted only
Browser 1.8/1.9 in the rank-10 switch and rejected other discriminators. Its
existing focused tests covered 1.8/1.9 but not 1.10. The uncached
`internal/trustedrunner` baseline test passed with Go 1.26.6.

**Dependencies and downstream impact.** Completed E22 supplies the published
Browser 1.10 count contract and remains immutable history. W8M W22.2 is blocked
on this source-owner repair; W8M W22.3 must bind the exact local P07 commit and
source digest and run fresh consumer smoke and qualification. W8M remains
responsible for its `session_posture: none` package-review input correction.
W22.4 adoption and W22.5's single read-only operation remain separately gated
by W8M acceptance. P07 does not change other package sources or grant target,
executor, adoption, push, or publication authority.

**Evolution.** No evolution pair is needed: P07 completes trusted-package
compatibility for the existing Browser 1.10 count direction and changes no
product boundary or public/private contract direction.

## Tasks

| Item | State | Notes |
| --- | --- | --- |
| P07.1 Implement Browser 1.10 trusted dispatch and exact restricted package check | `[+]` | Commit after focused tests, repository gates, and the exact restricted package lifecycle pass. The independent package report and source-byte comparisons are recorded below. |
| P07.2 Fresh v4 qualification, independent verification, and bounded review | `[~]` | From the exact clean P07.1 commit, run fresh v4 integration, loopback, journey, and three-repeat native qualification. Independently verify source/report bindings and confirm retained E22 reports still verify with unchanged bytes. Review the complete P07 diff beginning at bounded-review iteration 1; repair any P1/P2-or-higher findings and pass the gate. Commit this row locally after its acceptance and checks pass. |

### P07.1 Verification

The source change adds `uws.browser.1.10` to the existing active-profile rank-10
mapping. Focused uncached `go test -count=1 ./internal/trustedrunner`,
`make fast`, `go vet ./...`, `make check`, `check-doc-memory`, and
`git diff --check` pass with Go 1.26.6. The memory check emits its normal
warning that the milestone changed without an evolution pair; the evolution
rationale above records why no new direction version is needed.

The fresh restricted W22 package check is preserved at
`/home/peter/.local/state/openudon/p07-package-20260928/attempt-02/prepare.json`.
Its input SHA-256 is
`9a90d3b679f77bfff294038a6f0f59438238c5fc13d0713420fe4ef08e367be3`, package
SHA-256 is `6f374ea4b80f786af23615975fa189c0345ec989f17de951644bd94b13ed4d43`,
quality SHA-256 is
`20f2c97a1db16a7145ce16e970b550e2da0f049be68850ce8ad14ae6c9fd3f29`,
manifest SHA-256 is
`sha256:43553e8fc9d331087c88259c584c03df4e22a9535aca6511b4ddd857b7a4bcf9`,
and qualification SHA-256 is
`sha256:a22c73cbc6912ee9b9ca1a9057337c6c601afe3d0420cf1640ca72ee9cbd1f69`.
All five gates pass, including quality/secret scan and trusted dry-run; the
report says executor not invoked. No browser ran. The run used
`session_posture: none` in `.icot/browser-sources.json` (SHA-256
`43a55405dc0e44857028d0671f719cadf555434686aaef2c1543d1690ddb79fe`).
Its browser profile SHA-256
`f2f48267b31d2efdd8828ff504f1a5d82013338406f7b783ceedb657363606c5` and
workflow SHA-256
`bb4b85bedce6ff8bddb89657ed5f09bd58f3ff47455c655df80b6734944d998f` match the
corresponding W8M candidate source files byte-for-byte. W8M's supplied checkout
and these source files were not modified. A preceding fresh CLI invocation
stopped before lifecycle gates because its optional expected-input digest was
passed with the wrong prefix; it is preserved at `attempt-01` and was not a
package or source failure. W8M still owns binding the corrected posture metadata
in its candidate.

## Acceptance

- Browser 1.10 uses the established rank-10/v10 trusted handoff without
  changing executor protocol, approval, authentication, or runtime behavior;
  Browser 1.8/1.9 remain compatible and unsupported profile versions remain
  rejected.
- The exact W22 workflow/profile passes OpenUdon package quality, prepare,
  and trusted dry-run using corrected session-posture metadata, with no executor
  or browser invoked and no supplied source modified.
- Fresh current v4 integration, loopback, journey, and three-repeat native
  qualification pass on clean exact P07 source; reports independently verify.
- Retained E22 reports verify with unchanged digests and bindings.
- The bounded review gate passes with no unresolved P1/P2-or-higher finding.
- Work remains local on branch `P07`, with one verified commit per task row;
  no push, publication, W8M runtime adoption, or live target operation occurs.

## Verification commands

Use the repository-pinned Go 1.26.6 toolchain and the normal OpenUdon current
v4 qualification commands. Preserve the original W8M and E22 evidence; write
new reports to fresh private output paths. Run focused trusted-runner tests,
the exact restricted package lifecycle check, independent report verifiers,
`make fast`, `go vet ./...`, `make check`, documentation-memory checks, and
`git diff --check`. Do not fetch, install, upgrade, or mutate supplied source
worktrees during qualification.
