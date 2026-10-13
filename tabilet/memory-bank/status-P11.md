# P11 — Browser output binding correction

**Approved P11 correction — 2026-10-13.** The user approved the proposed
OpenUdon:P11 browser output binding correction and its planning, local execution,
bounded publication and ordinary published-module acquisition scopes. Source:
W20 browser-binding review F01; source priority not supplied, local P2, confirmed.
Published M33/C10/P10 tests show named `result_count` output binding compatible
in UWS but P10 assessment incompatible (`binding.output_reference_incompatible`
and `binding.output_missing`). Revalidation: Kinet
`8d318aab9f435fe65e716366208511bcd23e43c3` with uncommitted W20.4 source2,
OpenUdon `78f8609d2e21976fa1311aa914cf2e4170b895ef` and exact published modules.
P11 is new work; accepted P10 history and all existing row outcomes/review
counters remain unchanged. This intake starts no closing review. Audit disabled.

**State:** P11.1 complete; P11.2–.4 pending. Whole review0/10.
**Source baseline:** `78f8609d2e21976fa1311aa914cf2e4170b895ef` before planning.
**Coordinator:** [Stage12](../../../kinet/docs/stage12.md).
**Depends on:** accepted/published OpenUdon:P10, Browsertools:M33 and UWS:C10.
**Downstream:** Kinet:W20 and Kinet:M52.
**Authority:** requirements belong in the [owning specification](milestone.md#p11--browser-output-binding-correction).
The approved goal request owns effective scopes; this status grants nothing.

## Tasks

| Item | State | Notes |
|---|---|---|
| P11.1 — Freeze named-output regressions | `[+]` | Reproduce exact M33/C10/P10 mismatch with immutable synthetic action bytes; operation and step exports and downstream consumers, missing/escaped/nested references and unchanged HTTP controls. Pure offline readiness first. |
| P11.2 — Correct native output resolution | `[ ]` | Align browser response expressions and assessment references with reproduced named output declarations. Preserve full values, numeric spellings, unknownness and generic HTTP semantics. Do not rename shapes or suppress diagnostics in Kinet. |
| P11.3 — Qualify compatibility and public closure | `[ ]` | Focused/race/vet/static/full offline checks, immutable historical readability and explicit generated-proof compatibility decision, public-only compiled/module closure and adapted frozen consumer fixture checks. No real browser or private-runtime imports. |
| P11.4 — Review, publish and hand off | `[ ]` | Persist bounded independent whole review; coordinator publishes reviewed source/closure by at most two fast-forward pushes under the exact scope, verifies ordinary published module revision/files/H1 and exact remote closure, consolidates and retires. W20/M52 adopt only accepted/published proof; no prior grant transfers. |

## Acceptance and verification

Use the owning specification. All four rows and whole review qualify before
closure. Publication verifies exact immutable revisions; no directory/workspace
substitution or changing consumer inputs. Producer fixture proof and real
consumer adoption are separate. Required unperformed proof prevents closure.
Audit disabled; no browser/live/deployment/general sudo/SSH authority.

## Review

Whole-milestone review:0/10,not started. Intake is not review1.

## P11.1 verified characterization — 2026-10-13

Exact immutable action SHA56b48d53… and independent actual M33 Build/Verify/C10
controls preserve named outputs and complete declarations.29subcases reproduce
P10 operation/step/workflow exports, typed downstream ownership, invalid/missing/
escaped/nested references and unchanged HTTP behavior through Build/Assess/Verify.
The incompatible expectations explicitly characterize the defect; P11.2 must
replace valid cases with corrected support. No production or frozen fixture
changed. Coordinator source review passes; focused combined checks pass3.261s
(4.38s wall), earlier focused1.919s and malformed.205s; formatting passes.
All are offline pinned checks with120s outer bounds. Whole review stays0/10.
