# P12 — Browser authentication timeout interoperability

**Stage/owner:** STG-12 Phase A; OpenUdon. Required.
**State:** P12.1 in progress in a fresh producer worktree; other three rows pending. Whole review0/10.
**Source baseline:** `ec2fa2666dad604a4512f8fff4155d269c8597b3`; coordinator captures actual integration ref/primary/original leasebase before dispatch.
**Approval:** Latest explicit human P12 approval on 2026-10-13: “I Approve this P12 plan and bounded publication/acquisition scope, then resume the existing goal with audit disabled”. Planning application and later execution/publication remain separate operations under the resolved request.

**Approved reconciliation.** Stage 12 browser authentication timeout reconciliation, 2026-10-13; F01.
Source priority and review baseline: not supplied. Local severity: P1;
disposition: confirmed. Revalidated at Kinet `5bbf67dd56d30b41c05ff26319986a7b3179267d`, including
the unfinished 33-file M52.2 source (sorted manifest
`b63df5e1bf5a4c1f432a80b0b7f4aa386957c3c184e30d14bccf2006a3c7337e`), OpenUdon `ec2fa2666dad604a4512f8fff4155d269c8597b3` and native Udon
`5bbf2924d85b9f13b89bb8c000cb1b12e6c56491`. Public `packagev3/execution_plan.go:96–98,169–174`
and `packagev3/browser.go:255–262` reject every operation timeout;
native `pkg/uwsprofile/execution_profile.go:245` requires an authentication
operation timeout in `(0,600]` seconds, and `pkg/execute/browser_preflight.go:136`
requires prior authentication for actions. No authenticated byte inventory
satisfies both frozen producer and native profiles. Accepted P10/P11 and M52.1
remain history/foundation; OpenUdon:P12 owns the public producer repair and
Kinet:M52 owns actual consumer adoption. This is approved intake, not a closing
review iteration or a claim that the repair passed. Audit disabled.

**Lineage:** Accepted P10/P11, accepted W20 and M52.1 foundation; underway M52.2 producer compatibility blocker. No completed history is reopened.

**Specification:** [owning milestone](milestone.md#p12--browser-authentication-timeout-interoperability) contains scope, acceptance, verification and action requirements; declarations are not duplicated here.

## Dispatch and lease boundaries

**Depends on:** Accepted/published P11, UWS:C10, Browsertools:M33. Native Udon `5bbf2924d85b9f13b89bb8c000cb1b12e6c56491` is a frozen read-only fixture input.
**Downstream:** Kinet:M52 → W21 → U15 → M53 → OpenUdon:A32; all five conditionals unchanged.
**Parallel-safe:** no. Producer correction precedes resumption of the preserved M52 row2 lease. One OpenUdon lease; coordinator sole ledger/shared-memory/Git writer.
**Write set:** Scoped public browser predicate/plan code, additive tests and ordinary contract/qualification docs plus isolated external native pure fixture. No generic/non-browser restrictions, old goldens, wire/schema versions, private imports, default manifests, shared memory or other status changes by a child.
**Contracts read:** Immutable accepted P11/C10/M33/native inputs and frozen M52 partial-source context. Record exact source/module/byte hashes; no changing sibling checkout.

## Tasks

| Item | State | Notes |
|---|---|---|
| P12.1 — Freeze the authored authentication timeout contract | `[+]` | Specify existing optional UWS operation Timeout, finite (0,600] seconds for exact auth call1.0/1.1 only; retain untimed public proof/history semantics and genuine native unsupported behavior. |
| P12.2 — Admit bounded browser authentication in public proof | `[+]` | Add the browser-specific bounded-auth predicate to supplement and execution-plan construction with complete existing hash bindings; preserve generic/non-browser/action/registration/step/workflow restrictions and prohibit timeout injection, new wires and private imports. |
| P12.3 — Prove public and pure native exact-byte compatibility | `[ ]` | Positive/negative Build/Assess/Verify/plan/approval/Authority fixtures plus isolated exact-byte native Compile/CheckSupported for authentication/read/write, zero effect callbacks; cover timeout tampering, numbers/extensions, old untimed history/output goldens. Label source fixtures separately from ordinary adoption. |
| P12.4 — Qualify, review and hand off the published producer | `[ ]` | Offline owner gates, affected races, public closure/no-private-import guard, persisted whole review (maximum10), bounded coordinator publication and independently verified exact source/module handoff; actual Kinet adoption stays M52 and physical qualification stays M53.4. |

## Acceptance and verification

**Acceptance.** Public Build/Assess/Verify, execution-plan derivation, approval
and Authority accept exact timed auth1.0/1.1 fixtures and reject missing, invalid,
nonfinite, zero, negative, oversized, inappropriate-profile and tampered timing
where the relevant contract requires refusal. Historical untimed public proof
stays readable/unchanged and native unsupported. Positive and negative tests
cover complete hash bindings, numeric preservation, extensions, output resolution
and all unrelated flat restrictions. A separate isolated external fixture feeds
the exact freshly built public workflow bytes to pure native Compile and
CheckSupported at accepted Udon5bbf292, with authentication plus read and write
leaves. Assert zero effect callbacks; do not invoke Run or any browser. Record
exact bytes/digests and native source/module identities. Label a pre-publication
source fixture explicitly: it is producer compatibility evidence, never ordinary
published-module adoption or Kinet consumer acceptance. M52 separately adopts the
exact accepted/published successor; M53.4 alone owns physical browser proof.

**Verification.** Use retained exact Go/C10/M33/Udon inputs, GOWORK=off and
offline credential/model-free fixtures. Focused Go tests target 30 seconds,
cap two minutes; affected races cap five minutes. Run owner `go test ./...`,
`go vet ./...`, `make check`, public-closure/no-private-import guard and
`git diff --check` once at closure within existing budgets. Check additive
goldens/history/output/numeric/extension regressions and isolated pure native
Compile/CheckSupported. No Chromium/Playwright, Xvfb, external supervisor daemon,
disk snapshot, new cache, dependency/tool download, sudo or live action.


## Execution policy

Use the unchanged governing Kinet GOAL and complete resolved approved request. Task commits/local-rebase-ff, coordinator-only integration/closure/Git, retained existing caches and disabled audit apply. The explicit bounded source/closure publication scope is not authority for browser, sudo, deployment, live effects or arbitrary SSH. P11/M56 ended scopes never renew. Consumer ordinary adoption waits for accepted and independently verified exact publication.

## Review

Whole-milestone review: 0/10, not started.

## P12 fresh execution dispatch

Coordinator captures originalleasebase9fa3bfc2d7cc77b8c621001025fb32d714dd45b5, integrationref`refs/heads/main`, primary`/home/peter/Workspace/openudon`, worktree`/home/peter/Workspace/openudon.goal/P12`, branch`goal/P12`. Approvedplanningcommits Kinetdea8917a/Open9fa3bfc qualify sourcebaseline only; no publication or implementation acceptance. Rootownsallledger/shared/Git. Fourrows/whole0 retain counters, exactcachedproducers, immutableoldfixtures and no effects. Row1freezecontract first; actualCode implementation waitsitsreview/commit. Audit disabled.

### P12.1 contract freeze and task review1 started

[Authored timeout contract](../../docs/p12-browser-auth-timeout-contract.md) freezes atSHA3be7b99ba08415151f9fe6c141f7819e445d5035dbe87459bdc85a62da75295a. Cached public readiness passes0.026s:373packages/35reachedmodules at exactC10/M33/APItools inputs and no privateUdon or retainedinternal imports. Only ordinarycontract doc changed; no implementation/test/protected/browsing/publication acceptance claimed. Root starts independenttaskreview1; row1[~], otherthreepending, whole0 and auditdisabled remain.

### P12.1 completed after task review 1

Independent read-only task review 1 passes with no P1/P2/P3 findings against the frozen contract SHA256 `3be7b99ba08415151f9fe6c141f7819e445d5035dbe87459bdc85a62da75295a`. The retained offline readiness output independently confirms 373 packages, 35 reached modules and zero private/retained imports; a coordinator repeat to retain that output passes in 0.139 seconds. This completes the contract/readiness row only. P12.2/P12.3 implementation and compatibility, final verification, publication and consumer adoption remain pending. Whole-milestone review remains 0/10; audit stays disabled.

### P12.2 implementation dispatch

Row P12.1 is committed and integrated locally at `4877ce1bde4f4cfe55480738af039f56f6069ed9`. P12.2 now owns the two scoped predicate sites and additive public regressions under the frozen contract. The original lease baseline remains `9fa3bfc2d7cc77b8c621001025fb32d714dd45b5`; whole review remains 0/10. Coordinator alone writes all ledgers/shared memory/Git. No browser, publication or acquisition has run. Audit disabled.

### P12.2 frozen; task review 1 started

The four-file public predicate/regression change is frozen for independent review. New timeout tests pass offline in 2.715 seconds; affected browser, numeric, extension, output and golden controls pass in 15.968 seconds. The initial supplemental test compile error (a nonexistent VerifiedPackage accessor) was corrected in the test; it never ran effects. Formatting and whitespace checks pass. Row P12.2 remains in progress; task review 1 starts now, whole review stays 0/10. Native exact-byte compatibility, final owner checks, publication and consumer adoption remain pending. Audit disabled.

### P12.2 completed after task review 1

Independent read-only task review 1 passes with no P1/P2/P3 findings. The shared predicate requires exact independently reproduced authentication call provenance, finite `(0,600]` timing and every other existing flat restriction. The four-file source manifest is unchanged; focused tests and affected historical/browser/output/numeric/extension controls pass. Current public API documentation and product/architecture/tooling pointers describe the implementation while keeping P12.3 native/approval/Authority evidence, P12.4 closure/publication and M52 ordinary adoption pending. Whole review stays 0/10; audit disabled.
