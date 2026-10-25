# P14 — Typed authoring cards and text conformance

**Stage/owner:** STG-13; OpenUdon.
**Planning approval:** Owner-approved full rescope on 2026-10-25 (approval: “I aapaprove”), following the earlier whole-stage approval. Planning only; no execution, commit or grant activation.
**State:** Pending; no task started. Review 0/10.
**Specification:** [milestone.md](milestone.md#stg13-p14).
**Depends on:** None.
**Source baseline:** `8433e78f1b08be7a7d9cce6cad987cce747283c8`; existing dirty planning/history was inspected and preserved. Reconcile live source and exact accepted producers before implementation.
**Ownership:** One serial coordinator; parallel-safe no. Each row is a coherent future task-commit unit.
**Authorization:** See AUTHORIZATION_REQUIREMENTS in the specification. Status grants no action; consumed producer prerequisites require independently verified publication.

**Review provenance.** Kinet Stage 13 plan review (`kinet-stage13.md`), 2026-10-25; source commit baseline not supplied. Review H4 (High/local P2), M1 (Medium/local Lower) and M6 (Medium/local Lower): the parent go.work can select changing siblings; Kinet routing provides no P14 input; Kinet internal/cardfixture/cards.go is a formatter/frontend test harness, not a replacement public card system. P14.1/.4 own the authoring-only contract and standalone proof. Revalidated at OpenUdon `8433e78f1b08be7a7d9cce6cad987cce747283c8` with the existing uncommitted Stage 13 planning inspected; source/history changes were not absorbed. This is review intake, not a closing review iteration.

## Tasks

| Item | State | Notes |
|---|---|---|
| P14.1 — Specify additive authoring cards | `[ ]` | Define the smallest public typed review-card contract over existing step/operation identities. Resolve schema version from current source and explicitly represent missing, invalid and unresolved values. Limit these cards to authoring-stage data; keep consumer authorization/confirmation contracts separate, using existing Kinet card fixtures as adapter evidence. |
| P14.2 — Implement validation and deterministic text | `[ ]` | Add public validation and stable text projection without product planning, credential handling or execution. Preserve exact identity and binding evidence and fail closed on unsupported/ambiguous cards. |
| P14.3 — Publish conformance fixtures | `[ ]` | Add versioned positive/refusal cases, deterministic-text goldens and compatibility coverage for existing public clients; expose fixtures for both product adapters. |
| P14.4 — Qualify and hand off exact producer | `[ ]` | Run GOWORK=off public-boundary, focused and closure gates; record go env GOWORK=off and selected/compiled module identities with no local-directory replacements in publication/adoption proof. Complete review and prepare exact ordinary publication; independently verified publication gates W22/M136. |

## Acceptance and verification

OpenUdon publishes an additive typed authoring-card contract and deterministic text representation usable by independent Kinet and Ramen adapters.

- Cards cover intent/outcome/outline, exact source and operation identity, mapped fields/bindings, unresolved placeholders, validation and approval review. Data alone grants no authority.
- New schemas/fixtures are versioned and additive; existing consumers and immutable fixtures remain valid. Deterministic text and small local mirrors need no shared UI repository or private executor import.
- Cards describe authoring-stage data, not a replacement authorization or confirmation protocol. Kinet retains its product-specific exact approval/receipt contracts; Ramen retains its own plan/apply approval. Publish adapter guidance demonstrating that distinction.

Planned verification: Schema/JSON/text conformance positives and refusals, existing public-history/API compatibility tests, make public-boundary and make check. Apply the [shared verification policy](../../../kinet/docs/stage13.md#verification-policy), including explicit `GOWORK=off` for adoption and closure checks. All implementation evidence is pending.

## Review

Review gate iterations: 0

Task reviews: not started. Whole-milestone review: 0/10, not started. Follow the package's existing persisted review and closure procedure; planning, terminal rows and successful builds do not establish acceptance.
