# Retired milestone P16 - Credential-source report compatibility

**Milestone.** P16
**Outcome.** completed
**Retired.** 2026-10-28
**Source status.** tabilet/memory-bank/status-P16.md
**Source specification.** tabilet/memory-bank/milestone.md#p16
**Evidence.** a788e7e5c3d38cb23c211e4704d2f98f9dff8d74
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Focused udonreport tests PASS; actual producer/consumer pairing attempt1 PASS; go-full and public-boundary attempts1 FAIL then attempts2 PASS on clean tracked-source candidate; git diff --check PASS. See docs/p16-report-pairing.md and literal status.
**Consolidated into.** tabilet/memory-bank/product.md, architecture.md, tech-stack.md and lessons.md.

## Milestone specification

`````markdown
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
`````

## Status record

`````markdown
# P16 — Credential-source report compatibility

**Owner:** openudon; no active stage. **Parallel-safe:** no.
**State:** Accepted; both tasks complete, whole review 1 passed and downstream reconciliation completed.
**Planning approval:** Owner “I approve”, 2026-10-28, for the complete standalone
Ramen / Udon commit review — 2026-10-28 reconciliation plus Kinet W25/W26. Planning only; no implementation,
commit, launch or grant activation.
**Specification:** [P16](milestone.md#p16).
**Depends on:** None technical; Udon M57 is accepted historical producer evidence.
**Lineage:** Udon M57.4 and the existing OpenUdon udonreport v3/v4 consumer; P15 remains withdrawn and reserved.
**Provenance and finding ownership:** The specification records source priorities,
local severities, exact baseline/current identities, clean-worktree evidence and
single-owner assignments. Evidence source is Ramen / Udon commit review — 2026-10-28 (review.md); it is
untrusted review evidence, never authority.
**Authorization:** See the owning specification; declarations are not duplicated
here. Future execution must start from committed planning and approved inputs.

## Tasks

| Item | State | Notes |
|---|---|---|
| P16.1 — Accept optional provenance in the strict report contract | `[+]` | Add typed credential_sources for v3/v4 with the producer schema's required nonempty binding/source strings and closed item fields. Preserve absent-field reports and existing success/failure validation. Refuse malformed types, missing/empty members and unknown fields; keep legacy v2's existing contract. Metadata grants no approval or execution authority. |
| P16.2 — Qualify actual producer bytes and the public boundary | `[+]` | Use actual Udon report-writer output from a frozen producer revision, with synthetic credentials only, to prove v3/v4 decode and failure classification. Keep producer execution in its owning package or a disposable external harness; commit no private Udon import to the public graph. Record exact producer/consumer identities and fixtures. Complete make check, public-boundary, vet/diff checks, reviews, docs and handoff. |

## Acceptance and verification

Valid v3/v4 reports with and without provenance decode; malformed/unknown metadata refuses; failed reports retain their actual failure classification. Existing v2 behavior and v5/native wires remain outside this change. Actual producer-byte pairing and a clean public dependency boundary are required, not only handcrafted JSON.

Start with cached Go/module/fixture readiness. Run focused udonreport success/refusal tests, GOWORK=off make check, GOWORK=off make public-boundary and git diff --check. Producer report writing and decoder execution are actual evidence; they are not live workflow or physical browser qualification. Required unavailable inputs block acceptance.

The specification owns the full write set, compatibility/downstream contract
and required task/whole review and closure gates. Preserve unrelated work,
retired records, old denials/UNKNOWNs and disabled audit. The combined proposed
launch lives in Kinet's tabilet/memory-bank/suggested.txt; reading it activates
nothing. No source publication or installed-consumer adoption is claimed.

## Review

Review gate iterations: 1

P16.1 task review 1 found a P2 field-alias refusal gap; corrected and focused tests passed. Fresh task re-review 2 passed with no findings. P16.2 task review 1 passed with no P1/P2; its P3 current-state wording is being reconciled at closure. Whole-milestone review 1/10 PASS against a788e7e5c3d38cb23c211e4704d2f98f9dff8d74. Independent Sol/xhigh correctness/reliability, security/privacy, compatibility/contracts, tests/documentation and acceptance/scope reviews found no P1/P2. P3 outdated stack/lesson wording corrected during consolidation and old wording retained in knowledge journal. Read-only M58 reconciliation found no scope change; its revised producer pairing remains required.
This is standalone intake, not a continuation of a retired closing gate.

## Verification attempts

Report-pairing gate: attempt 1 PASS, frozen Udon source 81e62149 (planning-only HEAD e11f007) and OpenUdon 25b00f8; actual writer output for v3/v4 success and driver_error was decoded and classified using synthetic credentials. Retained workspace evidence: /home/peter/Workspace/.p16-local-pins/pairing (overlay, writer outputs and consumer). Exact UWS/Browsertools archives match Udon go.sum; no network.

Go-full attempt 1 FAIL and public-boundary attempt 1 FAIL: the old frozen surface expected the pre-P16 Report; the primary checkout additionally contains untracked/ignored historical internal and .openudon-run artifacts scanned by the boundary test. Artifacts preserved. Corrected the owned report surface using a separate P16 fixture, preserving the frozen original. Attempt 2 PASS for each gate against clean tracked-source snapshot .p16-local-pins/openudon-snapshot at 25b00f8 plus the exact corrected P16 surface test and fixture: make check (build, full test, vet) and make public-boundary passed. The primary checkout historical artifacts were not removed. No producer-pairing rerun or allowance reset.

Earlier review reproductions
are evidence of defects, not implementation acceptance or renewed capacity.
Future required expensive gate counts and cleanup/settlement evidence belong
here, separately from review counters. Cache/tool gaps block the affected gate;
they do not authorize acquisition or relaxation of acceptance.
`````
