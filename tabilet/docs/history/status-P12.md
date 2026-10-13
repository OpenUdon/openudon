# Retired milestone P12 - Browser authentication timeout interoperability

**Milestone.** P12
**Outcome.** completed
**Retired.** 2026-10-13
**Source status.** tabilet/memory-bank/status-P12.md
**Source specification.** tabilet/memory-bank/milestone.md#p12--browser-authentication-timeout-interoperability
**Evidence.** 54cd50d0b4a688fc1a2a32e679b4e6befc0f5114
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Focused public/native zero-effect compatibility, owner Make/race/vet/static/module/tidy/compiled closure PASS; exact independently published Git/ZIP/extracted source and native H1 PASS. docs/p12-qualification.json, docs/p12-compatibility.json and docs/p12-publication.json retain evidence. Normal closure publication checked afterward before dependents.
**Consolidated into.** tabilet/memory-bank/product.md, architecture.md, tech-stack.md, lessons.md; pending A32 public preservation and Kinet M52 consumer handoff.

## Milestone specification

`````markdown
### P12 — Browser authentication timeout interoperability

**Stage/owner.** STG-12 Phase A; OpenUdon. Required correction within approved R65 lineage, latest P12 approval 2026-10-13.

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

**Goal.** Make freshly authored, bounded authentication bytes pass the public
browser package lifecycle and the unchanged native authenticated preflight.

**Scope and compatibility.** Freeze the existing optional UWS operation
`Timeout` field: only exact browser authentication call profiles
`uws.browser-authentication-call.1.0` and `.1.1` may carry an authored, finite
value in `(0,600]` seconds. Untimed historical packages retain their previous
public Build/Assess/Verify and readable-history semantics; missing authentication
timeouts remain genuinely unsupported by native Compile/CheckSupported. Never
inject a timeout or rewrite approved bytes. Add a browser-specific bounded-auth
predicate for supplement and execution-plan construction; keep the generic
flatOperation predicate and non-browser, browser action, registration, step,
workflow, control-flow and extension restrictions unchanged. Bind authored
timeouts through the complete existing source/shape/workflow/package/assessment/
plan/approval/Authority hash chain. Add no wire/schema version or private import.
Preserve existing numeric lexemes, extensions, output bindings, goldens and
qualified default/frozen consumer pins; use separate additive regression fixtures.

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

**Dependencies.** Accepted/published OpenUdon:P11 at ec2fa2666dad604a4512f8fff4155d269c8597b3,
UWS:C10 and Browsertools:M33; accepted native Udon5bbf2924d85b9f13b89bb8c000cb1b12e6c56491
is an immutable read-only fixture input, not a new producer dependency change.

**Downstream impacts.** Kinet:M52 (actual adoption), Kinet:W21 (fresh
re-capture/conversion), Kinet:U15 (complete bound display/history), Kinet:M53
(exact final consumer/native pair before physical proof), OpenUdon:A32
(preserve the public correction). The existing five conditionals are unchanged.

**Parallel-safe.** no. This required producer correction precedes M52.2 resumption.
Use one external OpenUdon P12 lease under the governing Kinet goal; coordinator
owns all ledgers/shared memory/Git. Freeze producer/native/consumer inputs;
never use another agent's changing checkout. Declared implementation write set:
packagev3 browser predicates, execution-plan construction, additive public
tests/ordinary contract and qualification docs, and an isolated external pure
native compatibility fixture. Children return status/shared-memory proposals;
coordinator alone applies them. Exclude generic profile semantics, private native
implementation, old goldens, shared files, other statuses and default manifests.

**Review/closure.** Persist the whole-milestone review counter, maximum 10;
initial count 0/10. Required offline verification and independent full review
precede coordinator source publication; acceptance/retirement and independently
verified exact closure publication precede consumer dispatch. Publication is
bounded by the separate explicit human scope below. Preserve P11's consumed
2/2 pushes and M56's ended scopes; stop on uncertain effects/targets/cleanup.
No deployment, live, browser or general SSH authority follows.

**AUTHORIZATION_REQUIREMENTS** (conditions only; the resolved request supplies authority):

```yaml
AUTHORIZATION_REQUIREMENTS:
  cli.local:
    via: goal-policy
    executor: assigned-agent
    scope: declared-offline-P12-implementation-and-verification
  git.commit:
    via: goal-policy
    executor: coordinator
    scope: P12-task-and-substantive-closure-commits-under-task-policy
  git.push:
    via: explicit
    executor: coordinator
    scope:
      repository: /home/peter/Workspace/openudon
      remote_name: origin
      remote: git@github.com-tabilet:OpenUdon/openudon.git
      ref: refs/heads/main
      mode: fast-forward
      invocation_limit: 2
      count_failed_and_cancelled: true
      changes: reviewed-P12-source-and-closure-only
      prerequisites: offline-checks-and-independent-review; exact-closure-before-adoption
```

**Published-input acquisition requirement.** Action `cli.local`, via `explicit`,
executor `coordinator`: read-only Git/ordinary Go VCS acquisition of exact
reviewed published P12 source and closure revisions, only via
`git@github.com-tabilet:OpenUdon/openudon.git` or
`https://github.com/OpenUdon/openudon.git`, in existing caches. Independently
verify exact refs/ancestry/source and ordinary Go ZIP/H1 before adoption.
No SSH shell or other dependency/tool download. This supplemental protected
transport condition narrows ordinary local verification; the resolved human
request supplies its explicit acquisition authority.

No unrelated dependency/tool download, Kinet remote action, force/history
rewrite, browser, sudo, deployment or live authority. Requirements are declared
only here; status and launch input do not duplicate them.

**Tasks/review.** [status-P12.md](status-P12.md), four rows complete; whole review 1/10 passed.

**Accepted source.** `54cd50d0b4a688fc1a2a32e679b4e6befc0f5114`, ordinary `v0.1.1-0.20261013211918-54cd50d0b4a6`; independent source/Git/ZIP/extracted/H1 proof passes. Normal retirement closure publication is verified separately before dependents. No physical browser or Kinet adoption acceptance follows. Audit disabled.
`````

## Status record

`````markdown
# P12 — Browser authentication timeout interoperability

**Stage/owner:** STG-12 Phase A; OpenUdon. Required.
**State:** All four rows complete; whole review 1/10 PASS with no unresolved findings. Exact source publication/module proof pass; normal closure publication follows before consumer dispatch.
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
| P12.3 — Prove public and pure native exact-byte compatibility | `[+]` | Positive/negative Build/Assess/Verify/plan/approval/Authority fixtures plus isolated exact-byte native Compile/CheckSupported for authentication/read/write, zero effect callbacks; cover timeout tampering, numbers/extensions, old untimed history/output goldens. Label source fixtures separately from ordinary adoption. |
| P12.4 — Qualify, review and hand off the published producer | `[+]` | Offline owner gates, affected races, public closure/no-private-import guard, persisted whole review (maximum10), bounded coordinator publication and independently verified exact source/module handoff; actual Kinet adoption stays M52 and physical qualification stays M53.4. |

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

Whole-milestone review: 1/10 PASS; no unresolved P1/P2/P3 findings.

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

### P12.3 fixture dispatch

P12.2 is committed and integrated locally at `9352d394bfd2dabc4a627ccf4ca827d06526c737`. P12.3 now owns complete approval/Authority and isolated pure native exact-byte authentication/read/write qualification. Public source is frozen at that commit; native Udon remains exact accepted 5bbf292. Source overlays are producer fixture evidence only. No Run, browser, external request or effect callback is permitted. Original lease baseline and whole review 0/10 remain unchanged; audit disabled.

### P12.3 frozen; task review 1 started

[Compatibility matrix](../../docs/p12-compatibility.md) and its [portable source/evidence record](../../docs/p12-compatibility.json) retain the exact external fixture, 10 workflow identities and all three focused outcomes. The corrected matrix passes in 4.452 seconds for auth1.0/1.1 at 0.25/30/600 seconds followed by native read/write. Public approval/Authority and actual accepted native Compile/CheckSupported use the same workflow/data/source bytes; all ten effect callbacks remain zero. Timeout successors refuse old proof/authority; untimed public history stays readable with genuine native invalid_plan. The first two cheap runs diagnosed an unsupported synthetic a11y/missing count-output fixture, corrected only in disposable sources. This is source compatibility evidence, not ordinary adoption. Independent task review 1 starts; row remains in progress, whole review stays 0/10. Publication and M52 adoption remain pending; audit disabled.

### P12.3 completed after task review 1

Two independent read-only lenses pass task review 1 with no P1/P2/P3 findings. All seven embedded sources, ten workflows and the canonical filename/hash aggregate match retained bytes. Native/C10/M33 ZIP H1, go.mod H1 and extracted source provenance are independently checked (685/396/351 files). The coordinator retained a separately inspectable cheap confirmation: all ten cases pass, zero effects, 6.648-second wall time, unchanged fixture. Earlier failed fixture diagnoses remain retained; no native semantics or historical source changed. This completes source compatibility only. Whole review remains 0/10; P12.4 owner qualification/publication and M52 ordinary adoption remain pending. Audit disabled.

### P12.4 closing verification selected

Rows 1–3 are committed and integrated locally; frozen candidate `8081e7cdf82d5714c5f70ce0ace36efc6c1b4dfc` includes the complete source compatibility record. Root now owns final offline owner/race/vet/static/compiled-public checks, whole review and the separately approved two-invocation source/closure publication envelope. The independently reviewed one-use check runner binds candidate/source/tool/artifact identities before and after each phase. Full Make will run once for this unchanged candidate and includes the complete owner test suite. Audit disabled; whole review remains 0/10 until verification passes. No push or browser action has run.

### P12.4 closing checks: readiness passed

Frozen candidate `8081e7cdf82d5714c5f70ce0ace36efc6c1b4dfc` resolves 91 owner modules and 373 public packages with zero replacements, recursive errors or forbidden imports. Module verification passes in 5.600 seconds; tidy diff is empty. The reviewed one-use fresh CLI build is attempt 1, runtime cap 120 seconds plus 5 seconds cleanup. Subsequent race/vet/compiled-public/full-Make phases retain their own one-use intent/outcome records; no unchanged failed retry is allowed. Source, refs and tool hashes are checked before/after each phase. Whole review remains 0/10 and publication 0/2; audit disabled.

### P12.4 scoped gates passed; full Make attempt 1

Fresh candidate CLI build passes in 18.290 seconds, vet in 13.061 seconds, scoped staticcheck in 12.900 seconds and affected race checks in 259.542 seconds (five-minute cap). The transient checker cache is removed. Compiled/dependency proof passes: 91 selected modules, 90 archives, 17,550 archive files, 16,931 extracted comparisons, 373 packages/35 reached modules/372 exports, 20 immutable goldens and unchanged public declarations; no replacements/private runtime/retained internals. All candidate/ref/tool/input integrity checks pass. Full Make attempt 1 now starts against the same candidate, runtime cap 1,200 seconds plus 5 seconds cleanup, using the reviewed shim and matching freshly built CLI. It includes full owner tests; no duplicate unchanged full-suite invocation is planned. Whole review 0/10, publication 0/2 and audit disabled remain.

### Whole-milestone review 1 started

All required offline gates pass against candidate `8081e7cdf82d5714c5f70ce0ace36efc6c1b4dfc`; full owner Make attempt 1 passes in 178.354 seconds with all 1,691 frozen files and ref/tool identities unchanged. [Qualification](../../docs/p12-qualification.json) records the complete checks, immutable fixture/compiled/module boundaries and zero-effect native compatibility. Coordinator starts independent whole review 1 before source publication. Prior row reviews/outcomes remain preserved; P12.4 remains in progress. Publication 0/2 and audit disabled remain.

### Whole review 1 passed; reviewed source publication next

Both independent full-milestone lenses pass with no P1/P2 or higher findings. Current status and private receipt summaries are refreshed; the Lower dashboard observations are resolved before publication, with no behavior change or replay. Full code, fixture, ordinary dependency/compiled closure and every retained check support the qualification record. Source publication invocation 1/2 is the next coordinator action under the explicit approved OpenUdon origin/main fast-forward scope. Row P12.4 remains in progress until exact ordinary source/module proof and handoff; normal retirement closure publication follows under invocation 2. M52 waits for independently verified accepted closure and ordinary input. Audit disabled.

### Exact source publication and ordinary module proof passed

Coordinator push invocation 1/2 settled successfully; an independent remote-ref resolution and exact revision fetch identify `54cd50d0b4a688fc1a2a32e679b4e6befc0f5114`. Ordinary direct Go VCS resolves `v0.1.1-0.20261013211918-54cd50d0b4a6`; all 1,692 Git/ZIP/extracted source files and native archive/go.mod H1 values match. GOSUMDB was disabled: no signed checksum-database proof is claimed. [Publication](../../docs/p12-publication.json) records exact provenance. All four rows and whole review 1 now qualify locally; coordinator consolidates and retires the complete status/specification. The remaining second invocation publishes normal closure, independently verified before M52 adoption. P11/M56 scopes stay ended; audit disabled.
`````
