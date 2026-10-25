# P14 — Typed authoring cards and text conformance

**Stage/owner:** STG-13; OpenUdon.
**Planning approval:** Owner-approved full rescope on 2026-10-25 (approval: “I aapaprove”), following the earlier whole-stage approval. Planning only; no execution, commit or grant activation.
**State:** P14.1–.2 complete; P14.3–.4 pending; whole review 0/10. Publication remains forbidden by the invoking no-push policy.
**Specification:** [milestone.md](milestone.md#stg13-p14).
**Depends on:** None.
**Source baseline:** `8433e78f1b08be7a7d9cce6cad987cce747283c8`; existing dirty planning/history was inspected and preserved. Reconcile live source and exact accepted producers before implementation.
**Ownership:** One serial coordinator; parallel-safe no. Each row is a coherent future task-commit unit.
**Authorization:** See AUTHORIZATION_REQUIREMENTS in the specification. Status grants no action; consumed producer prerequisites require independently verified publication.

**Review provenance.** Kinet Stage 13 plan review (`kinet-stage13.md`), 2026-10-25; source commit baseline not supplied. Review H4 (High/local P2), M1 (Medium/local Lower) and M6 (Medium/local Lower): the parent go.work can select changing siblings; Kinet routing provides no P14 input; Kinet internal/cardfixture/cards.go is a formatter/frontend test harness, not a replacement public card system. P14.1/.4 own the authoring-only contract and standalone proof. Revalidated at OpenUdon `8433e78f1b08be7a7d9cce6cad987cce747283c8` with the existing uncommitted Stage 13 planning inspected; source/history changes were not absorbed. This is review intake, not a closing review iteration.

## Tasks

| Item | State | Notes |
|---|---|---|
| P14.1 — Specify additive authoring cards | `[+]` | Added authoringcard v1 types, separately embedded schema and docs/authoring-card-v1.md over existing source/selector spelling. Missing/invalid/unresolved states, literal null and opaque bindings remain explicit; no approval object. Offline Go1.26.6 focused schema/type check passed (0.015s); task review passed. Kinet internal/cardfixture remains adapter evidence only. |
| P14.2 — Implement validation and deterministic text | `[+]` | Added bounded strict Parse, Card.Validate and deterministic RenderText. Refuses unsupported/ambiguous identities, duplicate keys/targets, conflicting payloads, stale valid claims, invalid UTF-8 and oversize input; exact literal tokens/bindings and incomplete data remain reviewable. Focused tests (0.032s), vet, format/diff and task review passed. Adapter guidance preserves admitted type/requiredness/effect/incompleteness evidence in each product's own exact review. |
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

Task reviews: P14.1 passed against the actual additive types/schema/docs. P14.2 passed against validation, deterministic projection, bounded/error/duplicate/drift paths and product review separation; empty nonnil raw literals were refused before acceptance. P14.3–.4 pending. Whole-milestone review: 0/10, not started. Follow the package's existing persisted review and closure procedure; planning, terminal rows and successful builds do not establish acceptance.

## Execution evidence

Captured source: `refs/heads/main` at `e71c79f98bff9eb95ce139ef6289cd7f2d3f769b`; initially clean. Protocol normalized SHA256 `035d86ec7a288b99f9baece26a1e3fba4a4a385925b72e3718baa23c3c09bd56` matches its registry; all four participating addenda are absent. The invoking approved goal supplies local implementation/test/task-commit authority only; no push, acquisition, live operation or milestone closure is inferred.

Focused P14.1: `go test ./authoringcard -count=1`, retained Go1.26.6, `GOWORK=off GOPROXY=off GOSUMDB=off GOAUTH=off GOTOOLCHAIN=local`, declared ordinary source cache in place and approved private temporary root; exit 0. `gofmt` and `git diff --check` passed. Expensive/protected attempts: 0; whole review: 0/10.

Focused P14.2: the same offline tool/cache policy, `go test ./authoringcard -count=1` and `go vet ./authoringcard`, exit 0. Includes 22 typed-card refusals, six strict-JSON refusals and repeated full projections preserving exact numeric/identity/binding data and escaping multiline/control input. No external effect, new dependency, acquisition or expensive/protected attempt.
