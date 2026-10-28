## Approved execution and authoring contract remediation — 2026-10-28

The owner approved the complete standalone reconciliation with “I approve” on
2026-10-28, including the explicitly selected OpenUdon consumer fix and existing
Kinet W25/W26. This approves planning-file actions only: no implementation
launch, commit/publication/live authority or audit activation is started.
Source: *Ramen / Udon commit review — 2026-10-28* (review.md), untrusted evidence.
Current revalidation used clean worktrees at:

- Ramen: b21f71bf80500643baefde82c17d810d7af83342.
- Udon: 81e62149d48ce17ee153d931af27143f0f30d304.
- OpenUdon: 3ed803c8593b06f8cb1b6d8962afc058e6da057f.
- Kinet: d51015adab8dd84980081d765a21ebb74dec5201; W25/W26 remain pending.

Retired stages/milestones, withdrawn IDs, old permissions/attempt quantities,
UNKNOWNs and review counters stay preserved. Intake is not a closing-review
iteration. Each new milestone starts its own review at zero.

**Combined serial order:** OpenUdon:P16 → Udon:M58 → Ramen:M150 →
Ramen:M151 → Ramen:M152 → Kinet:W25 → Kinet:W26; seven milestones /18 tasks.
Technical edges are P16 → M58 and M150 → M151 → M152. W25/W26 keep their
existing three tasks and relative order; the other ordering is scheduling,
not a new technical dependency. Every milestone is parallel-safe no.

The single disposable launch reference is
[Kinet suggested.txt](../../../kinet/tabilet/memory-bank/suggested.txt).
It is proposed input, not approval or a review ledger. New grants require a
later matching-checksum launch approval. The dangerous environment preserves
local task commits, automatic within-scope refresh and workspace-contained local
verification, with EXTERNAL_MUTATIONS: none. Old completed-launch grants end.

**Current openudon actionable horizon:** P16.
Earlier completed/withdrawn sections below retain historical outcomes rather
than dispatch priority. Conditional unrelated work remains untriggered.

| ID | Status file | State | Dependencies |
|---|---|---|---|
| P16 | [Credential-source report compatibility](status-P16.md) | Planned; 2 pending tasks; review 0/10 | None technical; Udon M57 is accepted historical producer evidence. |

<a id="p16"></a>

### P16 — Credential-source report compatibility

**Owner/state.** openudon; approved planning, 2 pending tasks in
[status-P16.md](status-P16.md). No active stage; **Parallel-safe: no**.
**Depends on.** None technical; Udon M57 is accepted historical producer evidence.
**Lineage.** Udon M57.4 and the existing OpenUdon udonreport v3/v4 consumer; P15 remains withdrawn and reserved.

**Review provenance.** Ramen / Udon commit review — 2026-10-28 (review.md), findings in source order F01–F15.
Source/current revalidation: Ramen b21f71bf80500643baefde82c17d810d7af83342, Udon 81e62149d48ce17ee153d931af27143f0f30d304,
OpenUdon 3ed803c8593b06f8cb1b6d8962afc058e6da057f; all relevant worktrees clean. The owner's
“I approve” accepts the complete standalone reconciliation proposal; it is
planning approval only. This intake starts no task/whole review counter.
F02 (source P1, local P1, confirmed) is owned by P16.1. Current evidence: udonreport/report.go Report/Decode and Udon cmd/udon/main.go writeExecutionReport. The actual current decoder rejects credential_sources as an unknown field.

**Outcome.** Accept the producer's optional credential provenance without weakening strict report decoding or treating metadata as authority.

**Write set.** udonreport/, focused public report fixtures/tests and operator documentation, and the owning memory-bank records. No private executor import, runtime pin, package/authority wire or v5 report change.

**Tasks.**

- **P16.1 — Accept optional provenance in the strict report contract.** Add typed credential_sources for v3/v4 with the producer schema's required nonempty binding/source strings and closed item fields. Preserve absent-field reports and existing success/failure validation. Refuse malformed types, missing/empty members and unknown fields; keep legacy v2's existing contract. Metadata grants no approval or execution authority.
- **P16.2 — Qualify actual producer bytes and the public boundary.** Use actual Udon report-writer output from a frozen producer revision, with synthetic credentials only, to prove v3/v4 decode and failure classification. Keep producer execution in its owning package or a disposable external harness; commit no private Udon import to the public graph. Record exact producer/consumer identities and fixtures. Complete make check, public-boundary, vet/diff checks, reviews, docs and handoff.

**Acceptance.** Valid v3/v4 reports with and without provenance decode; malformed/unknown metadata refuses; failed reports retain their actual failure classification. Existing v2 behavior and v5/native wires remain outside this change. Actual producer-byte pairing and a clean public dependency boundary are required, not only handcrafted JSON.

**Verification.** Start with cached Go/module/fixture readiness. Run focused udonreport success/refusal tests, GOWORK=off make check, GOWORK=off make public-boundary and git diff --check. Producer report writing and decoder execution are actual evidence; they are not live workflow or physical browser qualification. Required unavailable inputs block acceptance. Required checks, task reviews, a passed
whole review within ten persisted iterations, documentation, downstream
reconciliation and normal retirement precede acceptance. No old attempt or
action quantity transfers. Cheap readiness/focused diagnostics precede each
counted expensive check; preserve the package verification-attempt gate and
record reservations/results separately from review iterations.

**Compatibility/downstream.** This extends the current source consumer only. No installed or pinned Kinet/OpenUdon consumer is automatically upgraded; source publication and ordinary module adoption are separately scoped. Udon M58 depends on this accepted contract for its final pairing.

**Authorization requirements (conditions, not permission).** The proposed
combined launch is local only. No publication, push, download, live origin,
sudo, SSH or deployment requirement is scheduled. Its named gates retain
separate bounded allowances only after a future checksum-approved launch.

```yaml
AUTHORIZATION_REQUIREMENTS:
  git.commit:
    via: goal-policy
    executor: coordinator
  cli.local:
    via: goal-policy
    executor: coordinator
    scope:
      repository: openudon
      task: P16-implementation-and-offline-verification
      gates: [go-full, public-boundary, report-pairing]
```


## Approved Stage 13 planning — 2026-10-25

The owner approved the revised **8-milestone / 34-pending-task** STG-13 plan after
the Kinet Stage 13 plan review (`kinet-stage13.md`). Owner-approved full rescope on 2026-10-25 (approval: “I aapaprove”), following the earlier whole-stage approval. Planning only; no execution, commit or grant activation.
Seven former cleanup milestones / 24 rows remain withdrawn history; HCL removal
and legacy-wrapper retirement are separate unnumbered candidates.
[Coordinator and acceptance](../../../kinet/docs/stage13.md); [stage overview](../../../kinet/tabilet/stages.md#stg-13).

**Current OpenUdon Stage 13 horizon.** P14 is accepted/retired after all four tasks and fresh whole2; no active OpenUdon Stage13 implementation remains. Kinet W22 and Ramen M136 own actual product adoption.

**Execution progress.** The separately approved local goal has completed P14.1–.3 and P14.4's local verification/readiness. Publication/ordinary consumer qualification and authoritative whole2 pass; the owner approved only the exact reviewed new-branch/acquisition scopes, not main pushes. The planning approval above supplies historical scope, not execution authority.

Preserve supported Ramen/Udon HCL input/output and supporting UWS codecs;
Ramen remains YAML-first. Stage 12 stays closed. Withdrawn Kinet:M54/M55,
Udon:M54, UWS:C11 and APItools:M86, accepted/retired evidence, all historical
UNKNOWNs and consumed grants remain unchanged. Stage 14 stays unscheduled;
audit stays disabled. No old permission, budget or trigger is reactivated.

| ID | Status file | State | Dependencies |
|---|---|---|---|
| P15 | [status-P15.md](status-P15.md) | Withdrawn; 3 rows `[X]`; undelivered; review 0/10 | None active; original dependencies retained in the withdrawn record. |

<a id="stg13-p15"></a>

### P15 — HCL-free producer adoption and public compatibility

**State.** Withdrawn by the owner on 2026-10-25; 3 untouched rows are `[X]`, review 0/10 unchanged. Withdrawn by the owner-approved 2026-10-25 rescope: retain supported HCL in Ramen/Udon and the supporting UWS codecs. This obligation is undelivered, not accepted. HCL removal and legacy-wrapper retirement move to separate unnumbered Candidate Directions; no replacement is scheduled and no active milestone depends on this row.
**Active dependencies/downstream.** None. M61 now owns product integration without this cleanup; it does not deliver the withdrawn removal outcome.
**Disposition and preserved task text.** [status-P15.md](status-P15.md). See the [unnumbered candidates](../../../kinet/tabilet/memory-bank/milestone.md#candidate-directions).

The following original specification is frozen planning history. Its dependency,
verification, execution-policy and authorization declarations are non-effective.

<details>
<summary>Original withdrawn specification (undelivered)</summary>

````text
### P15 — HCL-free producer adoption and public compatibility

**Stage/owner.** Kinet STG-13; OpenUdon. **State.** Approved planning; 3 pending rows; review 0/10.
**Depends on.** [UWS:C13](../../../uws/tabilet/memory-bank/status-C13.md), [APItools:M92](../../../apitools/tabilet/memory-bank/status-M92.md), [OpenUdon:P14](../docs/history/status-P14.md)
**Downstream.** [Kinet:M61](../../../kinet/tabilet/memory-bank/status-M61.md)
**Outcome.** OpenUdon consumes the HCL-free public producers while preserving typed cards, exact approval/handoff metadata and readable public history.

**Write set.** OpenUdon public UWS/APItools adapters, P14 schemas/fixtures where additive adaptation is required, module pins and compatibility documentation. Package-local documentation and status follow the owning change. Other repositories have their own owners. **Parallel-safe: no.**

**Lineage.** New OpenUdon consumer owner for STG-13; accepted/retired A32 and P13 remain frozen and supply no new grants.

**Acceptance.**

- Published C13/M92 are adopted ordinarily and P14 consumers remain compatible at their declared versions.
- Retain historical package/question/approval readers and deterministic review evidence while refusing new UWS HCL input/wrappers. Do not recreate retired private authoring or CLI code.

**Verification.** Public SDK/schema/history/card/approval tests, HCL/wrapper refusal, make public-boundary and make check; no private runtime imports. Preserve existing package verification standards, attempt limits and review gates. Use focused checks during coherent changes and required closure checks; reuse valid evidence, rerun affected checks after fixes and distinguish executed, mocked, compile-only and deferred proof. No implementation tests have run under this plan.

**Execution policy.** Future launch proposes workspace-contained local work, task commits, corrective-planning auto and option-ignore. Planning activates no grants. No live model/provider/browser, sudo, SSH, deployment or real user-ledger access is implicit. Substantial necessary discoveries use amendments; external testing-tool repair remains optional. No old budget, authority or historical UNKNOWN is reset.

```yaml
AUTHORIZATION_REQUIREMENTS:
  git.commit:
    via: goal-policy
    executor: coordinator
  cli.local:
    via: goal-policy
    executor: coordinator
    scope:
      repository: /home/peter/Workspace/openudon
      task: declared-implementation-and-verification
  git.push:
    via: explicit
    executor: coordinator
    scope:
      repository: /home/peter/Workspace/openudon
      remote: git@github.com-tabilet:OpenUdon/openudon.git
      ref: refs/heads/main
      mode: fast-forward
```

**Task/review owner.** [status-P15.md](status-P15.md) owns the 3 rows and persisted review; [the coordinator](../../../kinet/docs/stage13.md) owns cross-package acceptance and order.
````

</details>

## Current Stage 12 follow-up order

[A32](../docs/history/status-A32.md) is accepted/retired after all original
obligations, four task review passes and fresh Sol/xhigh whole review1. Source
d339d6879eee21b82e31d39e88ed27dd89299a04 is published/independently verified;
normal closure publication is the remaining coordinator handoff operation.
No active OpenUdon milestone remains. The cross-package goal has only five
conditional milestones/nineteen pending rows; deployment and named inventory
triggers are absent. Public SDK/API/history and consumer pins remain unchanged.
M53 early-use and M58 findings closures are preserved; no implementation
successor, browser/runtime/adoption/deployment authority or audit activation
is created. Earlier dated checkpoints below remain historical.

# Milestone

**Accepted P13 public producer — 2026-10-17.** Three rows pass task reviews1/1/1
and fresh Sol xhigh whole review1, no unresolved findings. Exact source
`5f2104652a7ef40e60cc04ce70e4b63206864f87` and ordinary module
`v0.1.1-0.20261017191049-5f2104652a7e` are independently published/verified;
complete1697Git/ZIP/extracted/nativeH1 and public compiled closure pass. Retired
P13 preserves all15 checks, including setup/red/umask failures. Normal closure
publication is pending before Kinet:M53.22 dispatch. M53.4 remains blocked at
physical failure3; its counters, unknowns and ended grants stay unchanged.
Remaining order: Kinet:M53 (adoption22, then original4/5) → OpenUdon:A32,
then the five unchanged conditionals. Seven active milestones/46 rows
(required2/27, conditional5/19); cumulative26/123 unchanged. Audit disabled.
Earlier dated checkpoints below are historical.

When `tabilet/stages.md` exists, it records stable `STG-` IDs, the current
stage, and provisional later ideas. Give each active milestone specification
its stage ID. An absent stages file means one implicit stage. Stage labels do
not determine task readiness or prove milestone acceptance.

Each active milestone specification defines scope, acceptance criteria, and
optional boundaries and parallel safety: `Depends on`, `Downstream impacts`,
`Write set`, `Contracts read`, and `Parallel-safe` (defaulting to `no`). A
milestone is parallel-safe only when unordered against other active work, its
write set is disjoint, and its inputs are frozen. Parallel safety never prunes
downstream impact reconciliation.

**Accepted P12 producer — 2026-10-13.** All four rows and whole review 1 pass. [Retired P12](../docs/history/status-P12.md) preserves the complete contract/tasks/reviews/outcomes. Authored authentication timing in exact call1.0/1.1 is finite `(0,600]`; historical untimed public proof remains readable and native unsupported. Public/private boundaries, complete existing hashes, old wires/defaults/goldens are preserved. Independent exact source and ordinary module proof pass at `54cd50d0b4a688fc1a2a32e679b4e6befc0f5114`; normal closure publication is checked before M52 adopts the exact successor. Physical proof stays M53.4; audit disabled.

Remaining order: Kinet:M52 → Kinet:W21 → Kinet:U15 → Kinet:M53 → OpenUdon:A32, then Kinet:M54? → Udon:M54? → UWS:C11? → APItools:M86? → Kinet:M55?. Ten active milestones / 43 rows: five required / 24 rows and five conditional / 19 rows. Approved total 21/92 (15 required/70, 6 conditional/22); earlier R65/P11/P12 planning counts remain historical. A32 remains gated by Kinet:M53. P11 pushes2/2 and M56 scopes stay ended; P12 source push1/2 settles, closure push2 follows. No deployment/live/browser/sudo authority or evolution version.

**Accepted P11 correction.** Four rows and whole review1 pass; exact closure `ec2fa2666dad604a4512f8fff4155d269c8597b3` is independently published and adopted by accepted W20. [Retired P11](../docs/history/status-P11.md) retains its complete proof and consumed publication2/2. Authentication timing is new P12 work.

## Stage 12 approved planning — 2026-10-09

**Approved reconciliation.** Stage 12 planning review, 2026-10-09; source priority
not supplied. Review baseline/current revalidation `477bf1a53591da70c98979a7a351db4fafac0ff9`,
including uncommitted Stage 12 planning files. F01/F02 are locally P1; F03–F09
are P2 (F03 partially confirmed, all others confirmed). F02 (P1, runtime owner Udon:M53) strengthens independent P10.4 browser evidence verification.
Owner status notes retain evidence and lineage. Intake starts no review iteration.
The coordinator now has 19 milestones / 84 pending rows (13/62 required, 6/22
conditional). Accepted local Kinet commits satisfy Kinet prerequisites; siblings
require exact acceptance and independently verified publication. Publication
grants remain ungranted launch prerequisites; no implementation is authorized.

Kinet R65 approved STG-12, browser profiles and legacy cleanup, as planning on
2026-10-09. OpenUdon owns two required units:

- **[P10](../docs/history/status-P10.md)** — Package v3 browser supplement.
- **[A32](status-A32.md)** — Legacy closure removal. It follows Kinet:M53's
  retirement of the pinned browser path.

No row has started. The M lane is full at M99, so new work uses the P and A
lanes. The shared contract is the [Stage 12 coordinator](../../../kinet/docs/stage12.md).
Planning grants no implementation, commit, publication or live authority.

**Historical Stage 11 context.** Stage 11 entries below retain their recorded
acceptance and order. Their references to provisional Stage 12 or no remaining
task describe those checkpoints. Current Stage 12 planning is above and in its
owner specifications; consumed Stage 11 grants do not carry forward.

## Stage 12 parallel dispatch policy

Approved Stage 12 parallel execution review, 2026-10-09; source priorities and
separate review baseline not supplied. Revalidation uses this package's unchanged
full baseline recorded above and includes uncommitted planning files. F01/F04
are local P2 workflow prerequisites; F02/F03/F05 are Lower scheduling findings
required by the owner-selected optimization. F03 is partially confirmed and
resolved by the selected early contract freeze; other findings are confirmed
(F05 conditional on existing operational authority). Intake starts no review.

One coordinator controls the integrated ledgers and serialized integration/closure.
The approved future launch uses `STATUS_PRIORITY`, `PARALLELISM: 3`,
`INTEGRATION: local-rebase-ff` and `COMMIT_POLICY: task`; these planning records
launch nothing. At most one live milestone per package and one in-progress row
per isolated lease. The nine eligible milestones are Kinet:A16, UWS:C10,
Browserdriver:M16, Browsertools:M33, OpenUdon:P10, Udon:M53, OpenUdon:A32,
Kinet:M54 and Udon:M54. All others remain sequential. Every dispatch checks
actual closure, publication where required, triggers and bidirectional contract/
write conflicts; priority positions and different repositories alone prove no
safety. Child scopes exclude shared memory, other statuses and launch input.

M51 freezes native-owner-reviewed report/host interfaces and conformance fixtures
before producers begin. P10 and Udon:M53 then implement independently; Udon
has no OpenUdon import. Their former sequencing/impact edge is replaced by
sibling compatibility verification at M56/M52, with both true producer inputs
retained. A32 and authorized Kinet:M54/Udon:M54 may overlap; C11 still joins
accepted/published A32 and Udon:M54 after clean authorized inventory.

Consumer checks read immutable exact snapshots, never another live checkout.
Shared-memory updates and combined integration checks are coordinator-only and
serialized. Leases rebase and re-verify the captured integration ref; affected
contracts require re-review without counter reset. A blocked lease retains its
branch/status and blocks dependents only; contract drift pauses affected leases.
Source publication remains separately named and ungranted. Audit stays disabled.
Kinet's adopted goal governs this cross-package run; sibling standalone protocols
and the serial Python API controller remain unchanged.

## Stage 11 post-acceptance remediation — 2026-10-08

The user approved the complete reconciliation proposal and its planning-file application. Five new package-local milestones / 28 task rows restore accepted supported contracts. Original Stage 11 acceptance and all completed records remain frozen; this intake starts no closing review.

**Serial order:** Udon:M49 → UWS:M09 → APItools:M83 → OpenUdon:M99 → Kinet:M49.
**Direct dependency graph:** UWS:M09 → APItools:M83, OpenUdon:M99, Kinet:M49; APItools:M83 → OpenUdon:M99, Kinet:M49; OpenUdon:M99 → Kinet:M49; Udon:M49 → Kinet:M49. Serial order is a scheduling gate, not an additional library dependency.

**Local owner:** M99 is accepted and retired after all three rows, full ordinary publication/consumer qualification and whole review3 at source801b45bb1aec297631b2e5d87beb1977788a1ab1. Exact pending Kinet:M49 contracts/closures are reconciled. The separately authorized same-ref normal retirement closure publication follows under the recorded grant. One serial execution owner remains required; source publication supplies no deployment/live authority.

**Coordinator:** [Stage 11](../../../kinet/docs/stage11.md#post-acceptance-remediation--2026-10-08). Exact M99 ordinary source/module and reviewed published evidence identities are established in current stack/qualification and the history record; Kinet independently qualifies actual worker/bundle artifacts before adoption acceptance. Installed M44 and independent browser/media/Phase A/legacy/frozen-consumer pins remain unchanged. Stage 12 remains provisional.

## Stage 11 active horizon

Approved 2026-10-06: both phases of [Kinet STG-11](../../../kinet/docs/stage11.md), with one serial execution owner across Kinet, UWS, APItools, Udon and OpenUdon. P09 is accepted after closing review 4 at qualified/public a6a3ef010fe27f277f8191204c1c81ea1cc0334b; Kinet U14 is accepted/retired review2; A31 accepted/retired review2; Stage11 owner horizon complete. M98 is accepted/retired after review 3 at qualified 08a3839f357ec40c7e50c8668e4bd7c8d86bb55a; its exact source/module publication and ordinary consumer proof passed. The history index preserves its complete record. [Specifications](#stage-11-cross-package-refactoring) below are the current horizon. Earlier completed horizons and records remain historical; planning grants no implementation or external authority.

## Stage 9 approved work — 2026-10-05

[M97](../docs/history/status-M97.md) is accepted and retired as this repository's Stage 9 broker handoff. Kinet coordinates `Kinet:M34 -> Udon:M46 -> OpenUdon:M97 -> Kinet:M35 -> Kinet:A14 -> Kinet:W14 -> Kinet:M36 -> Kinet:U12 -> Kinet:M37`; this ledger owns its tasks and closure. No implementation, acceptance or publication is established by planning. Future exact upstream revisions remain unset until observed.

## Current State

Stage 5 M91 extraction is complete, accepted and published at qualified
application source `3fd40d3f874bdcf668a018550112a02cd0d02409`, through
review/source publication `c8f2de71d983bea93dc1c045568c397a10a4eb56`.
Review 1 passed; required frozen integration and three fresh native repeats
passed, with retained fixtures/assets and historical locks unchanged.

M92 is also accepted/published: public UWS 1.12, package pending steps and pure
simulation qualified at `96c16acacc7f442858dac8a0fcb36c84991ebddf`; review4
passed, final integration and exact earlier native39 context independently
verified. M93 is also qualified, human-accepted and published at application
`f1273b622445d60dc7f3ea849e5b7f1a1f1e733a` through
`d5b483afc93dc5b25ac319b1ce590f5d4d6fd682`; native39, integration17 and both
visible journeys passed, review1 passed. M94 catalog discovery/provisioning is
also accepted/published at application/test source
`ee49fe433a4d476f8d28d3d888352293c490dfc6`, through
`5a2a2643ff71831ae72cdeca57baf197c1afbf16`; full owner/native conformance and
review1 passed. M96 is accepted/published at qualified application `eed683f27d448ca96af90e7bc5987967a6cd0335`, through verified source/review publication `d77f6d51262d0f311910070bc4a43f662260cd7e`; review1 passed and exact consumers are reconciled. Kinet M19/U07 and W8M W28 are accepted; Gate5B is approved. M95 removal is
accepted at final application `c2f161d762bc9f2217bbf0c34b00cdef64b0f7d0`,
publication `565f101439ffde86e4c4ef9e7a7fbbc1d16b4002`; review3 passed.
Kinet M20 then W8M W29 own final exact-pin consumer adoption.
Resolve completed prerequisites through the [history index](../docs/history/index.md).

E24 completed [actual namespace binding](../docs/history/status-E24.md).
E23 completed the current-stack/external-module browser-free input helper; see
[its reviewed history](../docs/history/status-E23.md). W8M owns cache eligibility
and fresh consumer acceptance.

M90 is complete and published for Kinet's stage 4 local execution slice.
Accepted implementation `ed5b206a524e6e193123b2d06714b75160379560` explicitly
selects report v5 and run-evidence v3, validates exact staged inventory and
preserves conservative unknown outcomes; legacy defaults remain unchanged.
The frozen real Udon M44 qualification and two-iteration bounded review pass.
Kinet W07 is reconciled to that exact published source in
`db35053f52fb7ee0e81d833752829b83f76ddd96` and remains the separately owned
consumer. Complete specification, status and publication evidence are in the
[M90 history record](../docs/history/status-M90.md). No runtime semantics,
UWS adoption, provider operation or deployment is included.

M87 is complete for OpenUdon-owned non-interactive step authoring commands and
published JSON conformance fixtures, consumed by Kinet W03 (sibling item S2a).
M87.1's v1 contract, schema, synthetic consumer fixtures, schema/UWS
ParamSchema conformance tests, and candidate/check/bind source-reference
continuity are complete. The owner selected one recursive UWS ParamSchema
object each for shared `inputs` and `outputs` on 2026-09-27. The actual C07.2
pending-wrapper mapping fixture remains in M87.6 and does not block
implementation. M87.2 step check, M87.3 step bind, and M87.5 flow-review are
complete. M87.4 is complete against the compatible published APItools M77
revision `v0.0.0-20260927073943-30c6f3bd5700` at commit
`30c6f3bd57001e521804a520e5aa21cb7dd95047`; workspace and standalone consumer
checks pass. M87.6 qualification and the persisted full-milestone review passed
at iteration 9. The full M87 specification, status, and review history are
preserved in the [M87 history record](../docs/history/status-M87.md). Kinet W03
remains the separately owned consumer; the authorized OpenUdon release commit
publishes the contract and commands it needs for its own real-tool check.
Contract work also reconciles APItools M77 and Kinet W03 consumer requirements.
M88 subsequently pinned APItools M78 `26bb05247d6c48f8ee60b9ae178f6ef6d48bbe3d` and corrected
step-authoring location, mapped-value, and credential checks; its record is in
[the M88 history](../docs/history/status-M88.md). Kinet W03 owns its real-tool
consumer check against the published OpenUdon revision. Simulation/browser
acquisition (S2b) and OpenUdon iCoT retirement (S3) are now planned in M91–M95; Authoring retirement remains deferred.

OpenUdon M89 was the approved Stage 1 remediation plan for package-local source
provisioning, explicit field mappings, and refusal diagnostics. It depends on
the accepted and published APItools M79 revision `e3625f6ef52ea54b7f78b7a4a4f1993bf8a06a46`
(`v0.0.0-20260928033144-e3625f6ef52e`); M89.1 pinned and verified that exact
source. M89.1–M89.5 are complete; the exact Kinet W04.6 consumer passed on
published OpenUdon revision `2e2ecedb32add53e10d6d81d4b2528ca2bdfdcc8`, and
the whole-milestone review passed at iteration 5. Its complete specification,
status, and review history are preserved in the
[M89 history record](../docs/history/status-M89.md). This remediation does not
reopen M87/M88 or promote S2b/S3.

OpenUdon's UWS 1.11 real-browser M86 qualification is complete: the pinned
sandboxed browsers launched, all three integration opt-ins passed, and the
current-stack loopback and journey suites passed at clean source revisions.
The exact commits, digests, and bounded review are preserved in the
[M86 history record](../docs/history/status-M86.md). This preservation does
not reclassify an earlier failure or authorize a public canary, live account,
or runtime adoption.

P08 completed the local OpenUdon maintenance for the Browser 1.10 runtime
handoff mismatch found by W8M W24.5. P07's completed package-dispatch result
remains historical; its v10 selection let restricted preparation pass but Udon
rejects Browser 1.10 actions under v10. W8M's sign-in and campaign-page request
completed, while its count action failed before extraction. P08.1 selects v11
and P08.2's focused synthetic journey and review pass. W8M retains
qualification, adoption and live-count authority.

E21.1 freezes the pre-E21 current scenario and integration locks for M86 v2
report verification. E21.2 advances the current scenario and integration
locks to the repaired Udon pin and a separate clean 14-source build closure;
its current report contracts are v3. E21.3 adds native v3 current-stack
qualification through the scenarios and BAP/BRP stages; historical native v2
remains the default and v1/v2 reports retain their readers. E21.4 is qualified
at OpenUdon `1007cdedf0acebf649bd3ddd065a0e42bac4f542`: current integration,
loopback, journey, and three-pass native reports independently verify with
exact clean source bindings. The bounded review is clear; E21.5 publication
completed when OpenUdon alone was pushed to `origin/main` at
`679f0bca630862a4ff46ea5a7edc0fc871abe938`. The accepted code commit is
`1007cdedf0acebf649bd3ddd065a0e42bac4f542`. Downstream W8M W21 qualification
passed independent verification and its separate feature branch was published
at `d1fd6c13d871622bf40466cba24649363d2f1846`; runtime adoption remains separate.

E22 adds the published Browser 1.10 count action and three synthetic
zero/one/multiple-row journeys to the explicit current stack. Scenario,
integration and native reports now use v4; historical v1–v3 readers preserve
their locks, manifests and gate inventories. Fresh v4 integration (19/19),
loopback (23/23), journey (14/14), and three-repeat native qualification
(39/39 stages) independently verify against clean OpenUdon
`9be9ff3f195ecaa8bdac88cc8616c5fc345dfeb3` and exact dependency/runtime
bindings. Bounded review iteration 3 passed with no P1/P2 findings. E22 is
complete; its full acceptance and attempt history are in the
[E22 history record](../docs/history/status-E22.md). This synthetic support
does not change W8M's adopted locks or authorize runtime adoption.

P07's reviewed preparation and qualification result is retired history; P08
owns the subsequent v11 executor-pairing correction. Their exact source,
verification and original review counters resolve through the history index.
W8M's retired W24 records its independently qualified P08.1 source and accepted
count with teardown. Stage 5 preserves that v11 behavior and grants no live
operation or adoption authority through historical records.

The UWS 1.11 real-browser M86, E15 registration-verification integration,
and E18 initialization-diagnostics integration milestones are complete. E15
and E18 closed against W8M W16.4i.39d synthetic qualification, independent
evidence checks and exact pass-one adoption. Their complete specifications and
status histories are in the [E15 history record](../docs/history/status-E15.md)
and [E18 history record](../docs/history/status-E18.md).

The original .34d `controller/worker_protocol` cause remains unresolved. W8M's
separate .34e live probe failed/consumed with `verification_timeout`, zero
application POSTs and unresolved provider cause; W8M real acceptance remains
incomplete and owner-scoped. These outcomes authorize no new browser, account
or live operation.

The history index holds 134 legacy-preserved status IDs and five normally reviewed completions. Legacy-preserved records retain exact source bytes and the frozen milestone text, but do not establish acceptance by themselves. Search the history by ID when needed.

## Active Milestone Specifications

### Stage 5 coordination (approved planning, 2026-09-30)

OpenUdon M91–M96 and Kinet M19/U07/W8M W28 are accepted published prerequisites.
M95 removed only its own iCoT transports and retained all23 capabilities;
remaining execution is Kinet M20, then W8M W29 final adoption. The
canonical cross-package text (order, Gate 5B, the W27 handoff rule, gates,
publication policy, and the F01–F10 / SR01–SR06 findings table) is Kinet
`docs/kinet-order.md` §6 (`../../../kinet/docs/kinet-order.md`); Kinet's
`tabilet/memory-bank/suggested.txt` is the only launch reference.

```text
Completed prerequisites: APItools M81/M80, Udon M45, Authoring M29, OpenUdon M91/M92/M93
Completed also: OpenUdon M94/M96; Kinet A10/W08/W09/W10; W8M W27
Completed also: Kinet M19/U07, explicit Gate5B, W8M W28, OpenUdon M95
Remaining: Kinet M20 → W8M W29
```

- M91–M94 are additive (stage 5A). They retain registration and
  authenticated/TOTP capture, and new packages use UWS 1.12.
- M95 removes iCoT only after Gate 5B and W8M W28's acceptance.
- M93.0 is an operation row: it prepares the private remote desktop under
  named authorization.
- Before consuming an upstream revision, record its actual accepted source,
  publication and qualification evidence. Never invent a future hash.

## Memory Bank Index

- This file owns milestones, work sequencing, acceptance criteria, the current-state dashboard, and
  the status-file index.
- Use [product.md](product.md) for product scope and non-goals.
- Use [architecture.md](architecture.md) for system boundaries and planned structure.
- Use [tech-stack.md](tech-stack.md) for dependency and tooling defaults.
- Use per-milestone `status-<LANE><NN>.md` files for task-level status history.

## Milestone families

Keep the original delivery outcome stable. Small directly related fixes stay in
the current task. Substantial required repairs belong in amendment records;
optional features, broader coverage and environment conveniences belong in
option records. An amendment must identify the original unmet requirement,
evidence and why a simpler implementation or verification route cannot meet it.
External testing-tool repair is optional unless the owner explicitly promotes
it, even under `CORRECTIVE_PLANNING: auto`. Mocks cannot prove unexecuted real
acceptance. Missing original requirements need repair or an owner disposition.

Each child owns its specification, task table and review in one file. Begin it
with a heading in the form `## <child-id> - <title>` and exactly one
`Parent: <root>` declaration. Keep concise parent/roadmap links instead of
duplicate specifications. Child format and numbering are defined in
[Status ID Pattern](#status-id-pattern). An initial option may be a short
finding without task rows.

Required amendments block the parent's acceptance; options do not. Preserve
completed work and review counters. Record deferred acceptance as `[!]` with
its amendment and resumption condition. `Parent:` does not mean a dependency
on the completed parent. A child's review never replaces the parent's review.
Keep detailed state in the owning file and Git history, not dated AGENTS prose.

For multi-milestone sequencing, family ordering and `OPTION_POLICY` follow the
selected goal protocol. Explicit owner selection is required for options;
`ignore` leaves them deferred throughout the goal. These policies supply no
extra action authority or budget. A claimed option still counts toward the
one-row ownership rule, even when the unscoped runner cannot execute it.

Retire child records under their original filename, preserving complete source,
`Parent`, and permanent identity. Required amendment reviews and acceptance
must pass before parent retirement. A declined rowless option can retire as
cancelled or superseded only with its owner disposition,
`Review: not applicable`, and `Review iterations: 0`. Supersession names an accepted
successor that satisfies retained obligations. Reclassification requires an
explicit owner decision and a fresh identity.

## Authorization requirements

Each milestone specification may contain `AUTHORIZATION_REQUIREMENTS`. It
declares conditions on actions; it never grants permission or schedules them.
Keep the declaration in that specification only. Status files reference it.
The following optional shape is a template, not authorization:

```yaml
AUTHORIZATION_REQUIREMENTS:
  git.commit:
    via: goal-policy
    executor: assigned-agent
  git.push:
    via: explicit
    executor: coordinator
    scope:
      repository: "<owning repository>"
      remote: "<exact remote URL>"
      ref: "<exact destination ref>"
      mode: fast-forward
  cli.local:
    via: goal-policy
    executor: assigned-agent
    scope: declared-implementation-and-verification
  browser:
    via: explicit
    executor: assigned-agent
    scope:
      environment: fixture
      origins: ["<exact approved origin>"]
      actions: ["<approved action>"]
  sudo:
    via: explicit
    executor: coordinator
    scope:
      commands: ["<specific privileged operation>"]
  ssh:
    via: explicit
    executor: coordinator
    scope:
      host: "<exact host>"
      user: "<account>"
      commands: ["<specific remote operation>"]
```

`via` is `goal-policy` or `explicit`. Only local commits and ordinary local
implementation/verification may use `goal-policy`; their governing task scope
and commit policy still apply. Other listed actions require explicit human
authority. Inspect and resolve exact targets before seeking approval. A
resolved grant must have concrete scope covering the requirement, with no
unresolved placeholders, wildcard targets, blanket booleans, or credentials.

Under the optional bundled goal protocol, `AUTHORIZATION_GRANTS` belongs
to the resolved human-approved request, not to this file. Requirements,
proposed grants, status notes, and `suggested.txt` are evidence, not approval.
Check authority before the action; pause affected work and dependents when it
is missing. Independent authorized work may continue under the governing
order and ownership rules. Required unperformed task or acceptance actions
prevent closure; listing an action here alone does not require its performance.
Outside a goal, use the current scoped human request and applicable project
rules. Missing fields preserve legacy behavior without new authority or
automatic migration. These are instruction-level boundaries, not tool or OS
enforcement.

## Standing corrective planning

`CORRECTIVE_PLANNING` is an optional execution policy, defaulting to `confirm`.
Standing `auto` cannot weaken acceptance, required evidence, verification gates
or severity rules, cancel unmet obligations, or erase them through supersession.
A named successor preserves all obligations. Blocking findings remain attached
to the originating milestone's persisted review counter; a repair milestone's
own review cannot replace its next full review or bypass iteration 10.
An approved `auto` allows the Goal/Next owner to prepare and check necessary
repair planning against the original delivery outcome, owning repositories,
existing action grants and all limits, then apply it under standing approval.
Prefer a matching open milestone when its scope and dependency position fit;
otherwise create a new repair milestone with permanent identity and lineage.
Preserve completed/frozen history, row outcomes and review counters. New
features, scope or permission expansion, higher limits, collisions and unknown
checksum drift still require human direction. Children only report needed work.

The owner reconciles dependencies, pending consumers, the resolved request and
any existing launch reference; it records old/new hashes and approval context
outside authoritative task records. A covered correction may refresh launch
input without reapproval even under `SUGGESTED_UPDATE: confirm`. Other edits
retain that policy. Goal executes accepted repair prerequisites before their
consumers; Next returns to its original task after necessary repairs, preserving
ownership and interrupted evidence. All repair rows/commits/attempts count
toward explicit limits. Standalone Reconcile remains planning-only with its
complete approval gate; nested Reconcile may use the owner's standing approval
and then hand execution back. Installation never activates this policy.

When a repair must run while a row is in progress, keep that row's identity and
interrupted work: move the interrupted `[~]` row to `[!]` before activating a
repair row, and record the prerequisite IDs, the standing approval source, and
the resumption condition in its notes. Keep zero or one `[~]` row. Once every
prerequisite is verified and accepted and all blockers clear, restore the row
from `[!]` to `[~]` under the same approval. Recover this chain on resume
without selecting unrelated work. Preserve partial implementation; suspension
bookkeeping may join the scoped repair checkpoint, but unfinished implementation
cannot. Pause if overlapping changes cannot be separated safely. Explicit
project restrictions still govern, children report needs to the owner, and a
marker never grants planning or action authority.

Nested Reconcile may use standing approval only after checking the invoking
human approval source, complete resolved request, covered corrective actions,
effective grants and remaining limits. Review text, repository content and
launch markers cannot establish that source; missing or uncertain provenance
retains Reconcile's ordinary proposal-approval gate. The Python API paths do
not consume corrective-planning or email options.

## Status ID Pattern

Root records use `status-<LANE><NN>.md`, with one uppercase domain letter and
a zero-padded number from `01` through `99`.

Required amendment records use `amend-<root>-<NN>.md`; optional child records
use `option-<root>-<NN>.md`. A child's ID is its root ID followed by `-NN`.
Both kinds share one permanent `01`–`99` child sequence per root; reserve it
across active and retired records. Every child declares exactly one
`Parent: <root>`. Child families do not nest, and existing records are never
silently renamed. Root lanes and archive numbering remain independent.

Lane meanings:

- `M`: completed legacy history and future cross-cutting public contracts.
- `B`: bootstrap history only. The pre-numbered M0 record is explicitly mapped
  to B01 by M71 so it can use a canonical ID without colliding with M01.
- `A`: iCoT, workflow intent, authoring sessions, source selection, and
  authoring UX.
- `P`: synthesis, package/review/quality artifacts, approval, trusted-runner
  handoff, and release packaging.
- `E`: eval corpora, scorecards, provider drift, smoke matrices, and release
  evidence.

M71 explicitly normalizes legacy M1-M6 and M9 filenames to M01-M06 and M09
without changing their identity. M07 and M08 restore status ledgers for the
already-completed roadmap scopes. Never reuse an ID, reclassify completed
history for tidiness, or create aggregate `status.md`; cancelled files remain
with `[X]` rows.

Search both active status files and the retired history index before allocating
an ID. A retired or cancelled ID remains reserved; an all-retired memory bank
remains initialized. A lane holds at most 99 IDs; open a new letter when full.
Do not rename existing IDs. Retain a consumed or superseded attempt as `[-]`
closed historical evidence only with its accepted successor recorded; never
retry it or count it as delivered acceptance. Context archive IDs, if any, use
an independent namespace.

Lane letters classify ownership rather than execution order. Independent A,
P, and E milestones may proceed together only when their sections name
non-overlapping files, resolved prerequisites, and downstream impacts. Shared
public contracts stay in M or explicitly name every cross-lane dependency.
Prefer one active implementation milestone per lane.

## Delivery Strategy

Keep OpenUdon as the public UWS authoring, review, package, and executor-handoff tool. Build in
verifiable slices: authoring and examples, deterministic synthesis, quality gates, eval and release
evidence, trusted handoff, readiness, and cross-repo compatibility. Push public workflow semantics
to `../uws`, API source metadata search/discovery/import/materialization/indexing to `../apitools`,
reusable execution to private executors such as `../udon`, and optional orchestration to
external services.

## Active And Parked Tracks

- The earlier horizon is complete; Stage 11 M98 is accepted; P09 is accepted after closing review 4; A31 is accepted/retired review2. M95 is accepted/published
  and retired; final Kinet M20/W8M W29 adoption remains in their own ledgers.
  APItools M81/M80, Udon M45 and Authoring M29 stay accepted prerequisites.

- Completed: M90 publishes explicit report-v5 handoff and strict v3 per-step
  evidence for Kinet W07; see the [M90 history](../docs/history/status-M90.md).
- Completed: M89's Stage 1 source-provisioning and mapping work consumed APItools M79 revision
  `e3625f6ef52ea54b7f78b7a4a4f1993bf8a06a46`; it is distinct from M87/M88's completed step-authoring
  contract. M89.1–M89.5 completed, the exact Kinet W04.6 consumer passed against published
  OpenUdon `2e2ecedb32add53e10d6d81d4b2528ca2bdfdcc8`, and the whole-milestone review passed
  at iteration 5; its record is in the [M89 history](../docs/history/status-M89.md). M88 corrected
  the confirmed Stage 1 binding findings, passed its
  review in iteration 2, and is retired in the [history record](../docs/history/status-M88.md).
  W03 is retired and its earlier consumer check remains historical. E22's Browser
  1.10 current-stack qualification and review remain complete; W8M's local W21
  candidate and any runtime adoption remain separate. The adopted W8M locks
  are unchanged. No deployment, public canary, or target operation is authorized.
- Completed: approved Stage 5 M95 and M96 are retired. M94 supplies accepted catalog discovery/provisioning; M93 supplies accepted supervised capture; M92 supplies pending/simulation; M91 retained all 23 journeys and extracted
  shared implementation; M95 removed terminal/UI/control/assets, retaining all23
  capability dispositions. E21/P07/P08
  completed normal history retirement before M91, preserving all original
  task outcomes, qualification evidence and review counts. W8M owns its later
  adopted runtime, consumed attempts and separately gated live operations.
- Parked: real-provider evidence, live W8M operation, and public canaries need
  separately approved scope and authority. Stage 5 M91–M95 delivered the
  approved subset of S2b/S2d and OpenUdon S3; remaining candidates stay below. W8M's authorized
  read-only operation remains gated by its own acceptance and is not part of P07.
- Completed history: use the single linked history index above for terminal
  ID records and the frozen earlier milestone text.

## Status Files

New Stage 13 records are indexed in [Approved Stage 13 planning](#approved-stage-13-planning--2026-10-25) above. Existing records below retain their earlier dispositions.

Active status files are indexed below. The memory bank remains initialized;
search the history index before allocating a future ID.

| ID | Milestone | Status file | State |
| --- | --- | --- | --- |

## Requested Changes After Initialization

For stage work, `memory-bank-propose --stages` requests creation or revision of
the optional `tabilet/stages.md` overview. `--stage STG-01` selects an existing
stage; the request must say when milestones should be planned. Equivalent
plain-English references are valid. Later-stage ideas may have only an ID,
tentative name, and brief intent. Additional context can be proposed later,
without assigning milestone IDs or task rows now. Stage IDs stay stable across
renames and reorderings and are never reused. Do not advance stages or schedule
later work merely because the current milestone closes.

For execution environment and authorization policy, `--environment dangerous`,
`--environment sandbox` (default when omitted), or `--environment least` sets the
hands-free stance in launch references and milestone policies, optionally modified
by expressions like `plus <action>`, `minus <action>`, or `but <action> in milestone <ID>`.

For a requested feature, candidate promotion, or future direction change,
inspect the current plan and implementation first. Separate the user's intent
from observed facts and unresolved assumptions. Propose the smallest pending
row and acceptance update that fits an existing owner; otherwise propose a
dependency-closed milestone with an unused permanent ID. Candidate triggers
prompt a fresh decision, never automatic promotion. Schedule by approved
priority and dependencies, not by review severity alone.

Present one complete proposal with outcome, owner rows, acceptance,
verification, dependencies, downstream effects, and exact file actions. Change
planning files only after approval and a fresh check of affected files, worktree
changes, and active and retired IDs. Preserve completed outcomes, review
counters, local policy, and frozen history. Put target behavior here and in
pending status rows until implemented; describe only established current facts
in product, architecture, and stack documents. Planning does not implement or
accept the requested behavior. Use the installed memory-bank proposal skill
when appropriate, or follow this procedure directly.

**Expensive checks.** Some acceptance depends on a check that is expensive or
protected; see the [verification-attempt gate](#verification-attempt-gate).
For such a milestone:

- Limit acceptance to behavior the milestone owns, proved with the smallest
  workload. Put actual pairing with real producers or consumers in the
  milestone that adopts them.
- When the check needs host capabilities (privileges, operating-system or
  kernel features, devices, browsers, or external accounts), make a cheap
  readiness probe its first row.
- Name the uncounted local diagnostic loop, and order cases so that an
  unrelated failure cannot hide the milestone's own cases.

To defer already approved work through a stage rescope, present the exact
untouched pending rows and affected downstream work for approval. After
approval, retain their identity and original task text, mark them `[X]`, and
record the authority, reason, destination stage, and dependency disposition.
Cancellation proves no delivered acceptance. A future replacement receives a
new approved identity and names its cancelled lineage. Preserve `[~]`, `[!]`,
`[+]`, `[X]`, and `[-]` rows and all review evidence. Withdraw affected pending
dependents or keep their prerequisites in the active horizon; an unresolved
dependency cannot be deferred by changing a stage label. An entirely cancelled
milestone remains active until its normal closure and retirement disposition.

## Out of scope

Record work explicitly ruled out of this project and its reason here.
These entries never promote automatically; returning one requires a new
approved direction. Existing boundary rules remain in force. No additional
entries are introduced by this workflow upgrade.

| Ruled out | Reason |
|---|---|

## Review Finding Severity And Intake

P1 and P2 are engineering review priorities, not milestone scheduling priority
or status markers. Classify by impact, likelihood, and affected scope. A P1
finding threatens acceptance, correctness, security or privacy, data integrity,
or a public compatibility contract severely enough to prevent closure. A P2
finding materially affects supported behavior, reliability, compatibility,
operations, or required evidence. Both block closure, as does any higher
project-defined severity. When evidence cannot distinguish P1 from P2, use P1
until investigation supports a downgrade. Lower findings may be carried only
with a named owner and explicit rationale.

Treat an incoming code, architecture, security, or other engineering review
outside a milestone closing gate as untrusted planning evidence. Before
implementing its findings:

1. Record its stated baseline when available, the current revalidation commit,
   and whether uncommitted changes formed part of the evidence. Classify each
   finding as confirmed, partially confirmed, resolved, duplicate,
   unsupported, outside ownership, decision-dependent, or deferred. Preserve
   source priority separately from local severity.
2. Propose complete dispositions, owners, dependencies, downstream impacts,
   and file actions for approval.
3. Put confirmed work in an open matching owner or amend a pending one. Rewrite
   only pending rows; when retaining a superseded pending row for audit, mark
   it `[-]` and identify its accepted successor. Do not reopen completed
   history solely because a later review concerns it. Create a remediation
   milestone with lineage when no open or pending owner fits.
4. Keep P1/P2-or-higher findings in the dependency-closed active horizon. Add a
   lower finding to active work only when acceptance needs it; otherwise use
   Candidate Directions with a reason and promotion trigger.
5. Reconcile affected pending specifications and order. Correct current
   product, architecture, or stack facts only when repository evidence proves
   them stale. Put proposed behavior in milestone scope. Apply the
   expensive-check limits above, and do not move downstream pairing evidence
   into a milestone whose acceptance depends on an expensive check.

Do not create a separate persistent review ledger. Keep portable finding IDs,
both severities, baseline, evidence, dispositions, and lineage in the affected
milestone/status notes. An incoming review counts toward the bounded closure
gate only when that gate was already active and the review was requested as its
next full pass.

## Verification-attempt gate

An expensive or protected check needs privileged setup, scarce hardware or
quota, a physical or live environment, explicitly granted actions, or minutes
per run. For a row whose acceptance depends on one:

- Reproduce only when safely possible within current authority and budget.
  Prefer the cheapest environment that can show the failure, such as an
  unprivileged local run on disposable fixtures. For unavailable environments,
  protected actions, exhausted attempt limits or non-reproducible incidents,
  record existing evidence and the reproduction limitation instead of forcing
  retries or inventing a failure. Required acceptance evidence and verification
  gates still apply; if missing, pause. Diagnostic runs are ordinary verification
  and never satisfy acceptance.
- Make each expensive run execute every case whose safety does not depend on an
  earlier result, and report all failures. Stop early only when continuing
  would be unsafe.
- Redaction protects real data. Raw diagnostics from synthetic, disposable
  fixtures may be kept in private, ignored artifact paths. Never commit them,
  and never treat them as evidence of effects, non-dispatch, retry eligibility,
  or authority.
- Each expensive run is one attempt. Persist the row's attempt count in its
  status notes; a session change does not reset it. Stop after three
  consecutive failed attempts at the same stage (or the limit the milestone
  sets), or when the failing stage lies outside the behavior the milestone owns.
  Mark the row `[!]` with a short diagnosis and options, such as a correction, a
  rescope or an environment change, and ask the user. Approval to continue sets
  a new explicit limit. Never repeat an unchanged failed attempt.

## Milestone Review And Closure

When the last open task in a milestone closes as `[+]`, `[X]`, or `[-]`, finish
the review before moving downstream. A `[-]` row is closed only when its notes
identify the consumed attempt or supersession and accepted successor. Terminal
rows alone do not prove acceptance.

1. Re-read this milestone's scope and acceptance. Verify the actual code or
   documents, including affected consumers and relevant compatibility,
   migration, rollback, security, concurrency, and failure paths.
2. Pin the milestone base commit and review the complete range, including
   staged, unstaged and relevant untracked changes under the commit policy.
   Review acceptance fidelity separately: missing/partial requirements, unasked
   behavior and incorrectly implemented requirements, quoting acceptance lines.
   An empty diff still requires acceptance review; do not invent changes.
   Cover all review lenses, combining/batching within host capacity. When an
   interactive request selects `MODEL_POLICY`, use the Goal/Next bundled
   reference for fresh read-only whole-milestone reviewers; the owner alone
   records results and fixes. Omission preserves host model settings. Persist the
   iteration number and findings in the active status notes before each pass.
   The first pass is iteration 1; an interrupted pass resumes at its persisted
   number, and session or runner limits never reset it. If no P1/P2-or-higher
   finding remains, the gate passes. Otherwise fix every blocking finding,
   rerun affected verification, and review the whole milestone again.
3. Stop after at most ten iterations. If iteration 10 still has a blocking
   finding, record `[!]` review rows with the findings and limit blocker. Do
   not start another automatic fix-review cycle or move downstream; ask for
   user direction. Carry lower findings only with a named owner and rationale.
4. Reconcile maintained product, architecture, stack, and lessons with the
   verified result. Check `tabilet/evolution/` under its existing trigger;
   create no version for an implementation-only advance. Revisit affected
   candidates; a satisfied trigger still requires a new approved proposal.
   Then run an environment retro on mistakes found during the milestone:
   mechanical issues become proposed deterministic checks; judgement issues
   become lessons or review rules. Include missing navigation or information and
   instructions that change nothing. Report these candidates by severity for a
   later approved proposal; create no unapproved task rows.
5. Reconcile active downstream dependencies and acceptance. During an ordered
   goal-protocol run, finish that protocol's downstream reconciliation
   before retirement. Consolidate knowledge and retire only after the review,
   verification, and downstream work pass.
6. Verify the resulting record and links, then commit substantive closure
   changes under the governing commit policy. Do not make an empty or
   redundant milestone-review commit. Report verification, review iterations,
   changed memory-bank files, any review commit, retirement record, durable
   lessons, and whether evolution changed.

## Goal lease cleanup

When a goal uses external lease worktrees, follow the selected goal protocol's
cleanup gate. Remove one only after recorded successful closure, matching
ownership, no active writer, verified integration and a clean worktree. Inspect
ignored data as well as tracked changes. Before freezing closure records,
preserve required qualification/recovery artifacts in an approved durable
location with their hashes and citations. Never rewrite frozen citations or
delete their required evidence; preserve and report uncertain leftovers.
This rule does not authorize cleanup of existing leases during an upgrade.

## Long-Term Memory And Retirement

Retirement is agent work during the milestone closure above, not a background
process or a separate archive run. Completed, cancelled, and closed-historical
rows remain in their active status file until the whole milestone qualifies.
Old completed milestones are not retired merely because this procedure is
adopted; an explicit cleanup request and adequate closure evidence are needed.

The owner approved one explicit legacy preservation migration on 2026-09-24.
It moved 134 status files whose task rows were all terminal into the history
index, leaving blocked E15 and E18 active at that time. Both later completed
under the normal reviewed procedure; see their permanent history records. The
source status documents retain
their exact bytes in individual records; the complete pre-migration milestone
text is frozen at
[`milestone-before-legacy-retirement.md.txt`](../docs/history/milestone-before-legacy-retirement.md.txt),
SHA-256 `26884eeda9af4ded30e6d545a33dca84c401d2320c22355fe4fb0336d5dcac71`.
The source HEAD was `71a4f78afbcf2180fc478ffa89c53544c9160648`, with
uncommitted memory-bank changes included in the frozen files. Each record
binds both documents by SHA-256. `legacy-preserved` records reserve their IDs
and preserve history; they make no passing-review, delivered-acceptance, or
dependency-satisfaction claim. Normal future retirement still uses the
reviewed procedure below. This one-time `legacy-preserved` migration cannot be
used for a new closure.
The separately tracked API runner validates the legacy envelope and frozen
snapshot at skills commit `dc674b876f0f93636f0e3154f731baeb199eb044`.

### Consolidate Before Retiring

Keep current facts in product, architecture, and stack documents. Maintain
[lessons.md](lessons.md) for concise, applicable, evidence-backed learning;
merge duplicates and retain useful lessons after their source milestone closes.
Before materially removing or superseding knowledge in those documents or
lessons, append its old wording to `tabilet/docs/history/knowledge.md` under a
unique dated heading. Include the source document and heading, reason,
supporting evidence, and replacement reference or reason none exists. Preserve
the old excerpt in a fenced Markdown block; append corrections rather than
rewriting old entries. Create and link the journal from the history index only
when needed. Routine wording edits require no journal entry. A context archive
is optional and never a retirement prerequisite.

### Retire A Closed Milestone

1. Require a passed review within the persisted ten-iteration limit, recorded
   verification, consolidated knowledge, and reconciled downstream work. No
   `[ ]`, `[~]`, or `[!]` row may remain. Every `[-]` row names its accepted
   successor. A cancelled or superseded milestone needs an authorized outcome;
   do not describe it as delivered acceptance. The legacy-closure and rowless
   option dispositions below are the only stated review exceptions.
2. Create `tabilet/docs/history/status-<LANE><NN>.md` with the envelope below.
   Preserve the complete final milestone specification and status document in
   separate literal Markdown fences, including all task text, notes, acceptance,
   and review evidence. Choose fences longer than any source fence. Relative
   paths inside the literal documents retain their source-document meaning.
   A child keeps its original amendment/option filename; both Source status and
   Source specification name its original child path. Preserve the complete
   child source, including `Parent`, specification prose and all earlier rows;
   only the selected closing row may change to completed.
3. Add one row to `tabilet/docs/history/index.md` with columns `Milestone |
   Outcome | Retired | Record | Summary`. Its ID, outcome, UTC date, and record
   link must match the envelope. Outcomes are `completed`, `cancelled`, or
   `superseded`; link to the record relative to that index.
4. Validate every envelope field and compare both retained documents with the
   complete active sources before removal. Use the full output of
   `git rev-parse --verify HEAD` for Git provenance; never use an abbreviated
   log hash. If validation fails, keep the active sources intact. Then remove
   the active status file, specification section, and status-index row; keep a
   single history-index link above. Repair maintained incoming links and
   downstream dependencies. Frozen archive snapshots remain unchanged.
5. Refresh an existing disposable goal suggestion to remaining active work, or
   remove it when empty. Do not create a new suggestion or launch execution.
   Verify the record and links before handoff. After interruption, reconcile
   active sources and destination before continuing; never overwrite a retired
   record or permit duplicate IDs.

The envelope records one field per line before the two literal sections:

`````markdown
# Retired milestone <ID> - <title>

**Milestone.** <ID>
**Outcome.** <completed, cancelled, or superseded>
**Retired.** <YYYY-MM-DD in UTC>
**Source status.** tabilet/memory-bank/status-<ID>.md
**Source specification.** tabilet/memory-bank/milestone.md#<original-anchor>
**Evidence.** <full Git HEAD or unversioned>
**Worktree.** <clean, includes uncommitted changes, or unversioned>
**Review.** passed
**Review iterations.** <1 through 10>
**Verification.** <commands, results, and supporting evidence>
**Consolidated into.** <current-document and lesson links, or no current-truth change>

## Milestone specification

````markdown
<Complete final milestone specification, not a summary.>
````

## Status record

````markdown
<Complete final status document, not a summary.>
````
`````

The evidence commit identifies the observed HEAD, not a claim that uncommitted
changes are in that commit. Without Git, both provenance fields are
`unversioned`. For cancelled or superseded outcomes, add `**Disposition.**`
with the authority, reason, and dependency disposition. A superseded outcome
also requires `**Successor.**` naming its accepted successor. Retired records
and index entries are frozen; append corrections to the knowledge journal or
open remediation work with a backlink rather than rewriting old records.

A legacy closure is a milestone that closed before the project adopted the
persisted review gate, so no review-iteration count was recorded. Retire one
only on an explicit cleanup request, with outcome `completed`, every row closed
and its original closure evidence available. Use `**Review.** legacy` and
`**Review iterations.** not recorded`, and add `**Legacy closure.**` explaining
that the milestone predates the gate and its literal status preserves the
original closure evidence. Verification records checks run at the evidence
commit without claiming a new review. Never use this path for a milestone that
closed after the gate was adopted, or to skip a required review. Existing
`legacy-preserved` migration records remain frozen and acquire no acceptance.

A declined rowless option may retire with outcome `cancelled` or explicitly
`superseded`, its owner's `**Disposition.**`, `**Review.** not applicable` and
`**Review iterations.** 0`. Preserve its complete source and permanent identity;
no implementation row is needed to withdraw a proposal. A superseded option
also names its accepted successor. This exception proves no product delivery.

Retrieve current work from the active memory bank first, then search the
history index by ID or topic. A stale status path resolves through the retired
record's permanent ID and original-path metadata; a missing ID does not permit
recreation. A completed record can satisfy a dependency only when its recorded
acceptance matches the required outcome. Cancelled or superseded records need
their disposition and successor checked against current implementation.
Retirement follows the existing commit policy, including the selected goal
protocol's explicit `COMMIT_POLICY: none` exception.

## Candidate Directions

Candidates have no lane, ID, status file, or execution-order entry until a
fresh scope and dependency review promotes them.

| Direction | Why Deferred | Promotion Trigger |
|---|---|---|
| Remaining S2b: live-read simulation and browser snapshot checks | M92/M93 now own pure simulation, pending packages and supervised capture; snapshot semantics and live-read execution are separately owned. | Owner contracts and explicit tier/snapshot adoption are approved. |
| Authoring S3 retirement coordination | OpenUdon M95 removes only its own iCoT and Authoring dependency; Ramen still uses generic icot/icotcli. | Ramen migrates and Authoring approves its compatibility/removal process. |
| Further package/source-family integration | A03/P01/A04/E01/E02 own the approved Browsertools authoring/evidence integration; other API/event source metadata remains owned by apitools and public semantics by UWS. | Another upstream contract is published and an OpenUdon-owned package/review outcome beyond this sequence is explicitly scoped. |
| Automated real-provider release evidence | Provider runs spend quota and can produce sensitive output; current policy remains local/manual. | Protected credentials, redaction, retention, spend bounds, and review-required CI policy are approved. |
| Trusted-runner capability expansion beyond P07 | OpenUdon hands approved packages to an external executor and must not absorb runtime semantics. P07 addresses the existing Browser 1.10 rank-10 dispatch gap without adding executor behavior; the per-step evidence gap was promoted as M90 on 2026-09-30 for Kinet's stage 4; other expansion remains here. | A further public handoff/evidence gap outside the existing rank mapping is demonstrated and scoped without importing private runtime behavior or weakening approval gates. |
| Execution-plan assertion hardening (stage11-siblings-review.md OpenUdon P3.4; source P3/local Lower; OpenUdon package owner) | packagev3/execution_plan.go assertions are protected by current closed schema validation; no reachable malformed panic was proved. | Schema relaxation or a reproducible reachable malformed case requires defensive projection checks and unchanged-wire regression qualification. |

## Notes

- Historical full real-LLM smoke baseline in README: 2026-04-28, `gemini-2.5-flash`, structured
  output path, ten original examples passed with zero legacy extraction fallbacks. Current local
  real-LLM defaults use `copilot-api` with `gpt-5.4-mini`.
- M35-M37 retain historical local release-candidate and product-smoke evidence.
  No corresponding public GitHub tag or release exists; M69 owns the first
  public tag as v0.1.0. Real provider outputs remain ignored and local/manual.
- Advisory n8n reducibility fixtures remain part of the eval corpus. They should not introduce
  n8n-specific runtime behavior into OpenUdon or udon; use explicit intent, OpenAPI, and generic
  `fnct` or control-flow modeling.
- Normal deterministic gates remain `go test ./...`, `go vet ./...`, `make check`, and
  `git diff --check`. GitHub Actions runs provider-free public module gates and the repository
  boundary guard with `GOWORK=off`; sibling readiness, `check-doc-memory`, and real-provider release
  evidence remain local/manual.
- `openudon run` is the only OpenUdon-owned path that invokes a trusted executor runner, and it
  requires approval JSON plus a valid handoff package.
- OpenUdon no longer imports udon as a Go module; udon is an optional external trusted executor
  behind the run-config handoff.
- After a major review or milestone, check whether [tabilet/evolution/](../evolution/) needs a new
  prompt/result version.

## Stage 11 cross-package refactoring

Approved review-intake amendment, 2026-10-06: 18 required milestones / 87 pending rows across the five owners. Scope and milestone IDs are unchanged; the coordinator records the approved publication proposal, which needs a separate execution-time grant. Full local source baseline remains `7cd7fbb837fb87e1ca4abea2a362790b0f434188`, including the reviewed uncommitted planning state. This intake does not start a closing review or authorize implementation.

**Approved source.** User-approved complete proposal, 2026-10-06; source baseline `7cd7fbb837fb87e1ca4abea2a362790b0f434188`. [Coordinated contract](../../../kinet/docs/stage11.md) defines both phases, cross-package order, compatibility and acceptance. The request to implement the proposal authorizes its planning files only.

One execution owner, serial execution and task commits under the later confirmed goal. Planning authorizes no code execution, commit, publication or external operation. Source publication requires separately named authority; a status marker or local build is not publication. Consumers must record exact accepted and published prerequisites before adoption. Default checks are offline, credential-free and model-free. No deployment, live ledger migration, real API/model/mail action or registration change.

## Stage 11 candidate dispositions

Consumer-owned non-browser drafting is promoted through Kinet W18 and OpenUdon M98/P09/A31. Browser-dependent retirement goes to Stage 12. Live-read tiers, browser snapshot semantics, real-provider evidence and the frozen Ramen/Authoring catch-up remain deferred.

## Stage 12 browser profiles and legacy cleanup

**Approved source.** User-approved complete STG-12 proposal, 2026-10-09 (Kinet
R65). Source baseline `477bf1a53591da70c98979a7a351db4fafac0ff9`, clean.
[Coordinated contract](../../../kinet/docs/stage12.md) defines the order,
triggers, compatibility and acceptance. The request authorizes planning files
only.

**Execution rules.** One coordinator, scoped eligible leases and serialized integration, with commit policy
from the later confirmed goal. Source publication requires separately named
authority. Consumers record exact accepted local Kinet prerequisites and accepted,
independently published sibling prerequisites before adoption.

## Stage 12 frozen consumer input checkpoint — 2026-10-09

P10's status records exact Kinet:M51.3 contract/fixture input at `589b844628b358167fe0a0137eded11c1028875c`.
M51 whole acceptance, producer qualification and independently verified
publication remain required at their existing gates. No row or review starts;
local-only source work grants no publication, browser or live authority.

**Accepted local M51 prerequisite.** [Retired M51](../../../kinet/tabilet/docs/history/status-M51.md) at observed task/review evidence `33980aafdc614424f61c32a2d79171689c7952bc` (reviewed implementation874c89e, declared coordinator closure) qualifies the unchanged frozen consumer manifest0c445a5c…. Browser qualification1+3 is consumed. This producer remains pending with its original row count/review counter; exact accepted publication and its own browser authority, where required, remain separate.

## Accepted UWS:C10 prerequisite — 2026-10-09

Exact public implementation f01a2542410c583d0ea909dadd8f17527cc0d27d resolves as v0.0.0-20261009220914-f01a2542410c, archive h1:TTjAn0++TdGHY0vahbXw3XsICQaavBytoqTSHu0K6o0= and GoMod h1:DlqFOnO9lbmYWLLIh5WicNX6NTWIuytU6mIHmxj9BVw=. Whole review1 and independently fetched closing publication 60e1baa0470cd7dca5d3166d344896bc0689abf7 qualify this input. See [public proof](../../../uws/docs/c10-publication.md). Own adoption, other prerequisites, custody and qualification remain pending; no task/review/pin change. Scoped source-publication authority is separate conversation approval; this record grants none. Audit stays disabled.

## M33 accepted source and next-unit checkpoint — 2026-10-10

Browsertools:M33 is accepted/published at1859f5e34367cb0a3fbc34ddbc2446b59ce3f2bc,
whole review2, full351-entry independent tree proof. The owning P10 status
records exact APIs, compatibility and required ordinary module acquisition.
The human explicitly approved P10/Udon:M53 execution; original dependencies,
rows and persisted review0 remain unchanged. The final M33 archive/sums remain
unacquired/unapproved, so adoption/implementation dispatch waits for that input.
No workspace/directory substitute, browser/live authority or audit activation.

## Ordinary M33 input qualification — 2026-10-10

ExactpublicM33 archive1859 andnativechecksums arequalified, all351files match
acceptedsource, offline tests/vet pass. Owning P10 status links completeproof.
Independent P10/M53 leases are dependency-ready under the approvedexistinggoal.
No task/reviewcounter or browser/audit setting changes at this input checkpoint.
