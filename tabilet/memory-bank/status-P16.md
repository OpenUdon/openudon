# P16 — Credential-source report compatibility

**Owner:** openudon; no active stage. **Parallel-safe:** no.
**State:** Both tasks complete; whole-milestone review and closure pending.
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

Review gate iterations: 0

P16.1 task review 1 found a P2 field-alias refusal gap; corrected and focused tests passed. Fresh task re-review 2 passed with no findings. P16.2 task review 1 passed with no P1/P2; its P3 current-state wording is being reconciled at closure. Whole-milestone review: 0/10, not started.
This is standalone intake, not a continuation of a retired closing gate.

## Verification attempts

Report-pairing gate: attempt 1 PASS, frozen Udon source 81e62149 (planning-only HEAD e11f007) and OpenUdon 25b00f8; actual writer output for v3/v4 success and driver_error was decoded and classified using synthetic credentials. Retained workspace evidence: /home/peter/Workspace/.p16-local-pins/pairing (overlay, writer outputs and consumer). Exact UWS/Browsertools archives match Udon go.sum; no network.

Go-full attempt 1 FAIL and public-boundary attempt 1 FAIL: the old frozen surface expected the pre-P16 Report; the primary checkout additionally contains untracked/ignored historical internal and .openudon-run artifacts scanned by the boundary test. Artifacts preserved. Corrected the owned report surface using a separate P16 fixture, preserving the frozen original. Attempt 2 PASS for each gate against clean tracked-source snapshot .p16-local-pins/openudon-snapshot at 25b00f8 plus the exact corrected P16 surface test and fixture: make check (build, full test, vet) and make public-boundary passed. The primary checkout historical artifacts were not removed. No producer-pairing rerun or allowance reset.

Earlier review reproductions
are evidence of defects, not implementation acceptance or renewed capacity.
Future required expensive gate counts and cleanup/settlement evidence belong
here, separately from review counters. Cache/tool gaps block the affected gate;
they do not authorize acquisition or relaxation of acceptance.
