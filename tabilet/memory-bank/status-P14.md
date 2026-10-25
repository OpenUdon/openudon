# P14 — Typed authoring cards and text conformance

**Stage/owner:** STG-13; OpenUdon.
**Planning approval:** Owner-approved full rescope on 2026-10-25 (approval: “I aapaprove”), following the earlier whole-stage approval. Planning only; no execution, commit or grant activation.
**State:** P14-W1 correction is locally committed and qualified; P14.4 is suspended pending P14-W2 now-authorized exact publication and remaining evidence/acceptance. Authoritative whole review counter remains 1/10; parent iteration 2 is pending. No general row is in progress; no gate pass or closure is claimed.
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
| P14.3 — Publish conformance fixtures | `[+]` | Added docs/fixtures/authoring-card-v1: initially 25 JSON/five text cases; the still-unpublished draft now has 33 JSON/nine text cases after the P14-W1 Unicode correction. Existing case bytes remain unchanged and the draft manifest binds the new helper/cases. New conformance/byte/depth checks, frozen public API/import checks and package/history/approval/handoff/wire checks pass. This is repository conformance data, not remote source publication. |
| P14.4 — Qualify and hand off exact producer | `[!]` | P14-W1 correction committed locally at e3bd5b07b1509e831be834f0e37aaba572f44975 after red/green regressions, extended corpus and fresh DEFAULT/xhigh fix review (no P1/P2; P3 resolved). Exact clean detached p14-unicode-qualification passes public-boundary (0.188s) and overall full check 2 (24.261s). New compiled authoringcard BuildID/source/corpus identities are bound in docs/p14-unicode-correction.json; selected 91/compiled 35 module identities remain exact with no replacements. Prior 9e1 check 1 PASS (26.069s), proof and archives remain unchanged. Whole counter remains 1/10 with parent iteration 2 pending; P14-W2 publication remains denied/unperformed and gates acceptance/W22/M136. |

## Acceptance and verification

OpenUdon publishes an additive typed authoring-card contract and deterministic text representation usable by independent Kinet and Ramen adapters.

- Cards cover intent/outcome/outline, exact source and operation identity, mapped fields/bindings, unresolved placeholders, validation and approval review. Data alone grants no authority.
- New schemas/fixtures are versioned and additive; existing consumers and immutable fixtures remain valid. Deterministic text and small local mirrors need no shared UI repository or private executor import.
- Cards describe authoring-stage data, not a replacement authorization or confirmation protocol. Kinet retains its product-specific exact approval/receipt contracts; Ramen retains its own plan/apply approval. Publish adapter guidance demonstrating that distinction.

Required verification: Schema/JSON/text conformance positives and refusals, existing public-history/API compatibility tests, make public-boundary and make check. Apply the [shared verification policy](../../../kinet/docs/stage13.md#verification-policy), including explicit `GOWORK=off` for adoption and closure checks. Implementation, conformance and local qualification are recorded below; independently verified publication, authoritative whole review and closure remain pending.

## Review

Review gate iterations: 1

Task reviews: P14.1 passed against the actual additive types/schema/docs. P14.2 passed against validation, deterministic projection, bounded/error/duplicate/drift paths and product review separation; empty nonnil raw literals were refused before acceptance. P14.3 passed against all 25 case outcomes, independently prepared text, hash-bound source/schema/cases, exact byte/depth controls and unchanged consumer history/API contracts. P14.4's fresh DEFAULT gpt-6.1-sol/xhigh review was limited to verification/input/publication evidence and the documentation-only follow-up; no P1/P2 evidence defect. P14-E1/P3 stale owning/shared summaries were corrected in the coordinator documentation commit; no evidence or acceptance changed. P14.4 acceptance remains pending. Authoritative whole-milestone review 1/10 completed with the P2 findings below; the earlier evidence-only task review was not a whole-implementation gate. Follow the package's existing persisted review and closure procedure; planning, terminal rows and successful builds do not establish acceptance.

### Authoritative whole review iteration 1 — completed with findings

Fresh read-only WHOLE_MILESTONE_REVIEW `gpt-6.1-sol/xhigh` reviewed the complete
candidate from integration baseline `e71c79f98bff9eb95ce139ef6289cd7f2d3f769b`
through `e4a0d4241b2b59d23044fdef35defeae5b593ed5`, including review-state updates.
Qualified production/schema/test bytes were `9e1a8c736ad9793aba613d26d1e8cd577c5bc267`.
The pass does not satisfy the gate; its persisted counter remains **1/10**.

- **P14-W1 — P2, correctness/contracts.** `authoringcard/validate.go:47`
  checks raw UTF-8 before Go JSON decoding, which replaces escaped lone UTF-16
  surrogates. For example, source IDs `items\ud800` and `items\udc00` collapse
  into the same decoded identity as intentional `items\uFFFD`. This violates
  “exact source and operation identity” and P14.2's exact evidence/fail-closed
  requirements. The reviewer established the decoder path from pinned Go's
  `encoding/json/decode.go:98–101`; no reproduction was executed by the reader.
  Owner: P14.4. Reproduce cheaply, refuse unpaired escapes in this new contract,
  retain valid pairs/literal U+FFFD and historical wire behavior, then verify.
- **P14-W2 — P2, unmet publication outcome/evidence.** The original outcome
  says OpenUdon “publishes an additive typed authoring-card contract”; there is
  no published module identity or independent publication proof. P14.4's local
  “prepare exact ordinary publication” sub-obligation is legitimately prepared,
  and actual Kinet/Ramen adoption belongs to W22/M136. Publication nevertheless
  remains an unmet producer outcome and independent consumer/full-stage gate.
  Owner: coordinator; explicit no-push remains binding. No action is authorized.

All five lenses were covered. No additional material defect or P3 was found.
The reader checked every new production/schema branch, all tests/corpus/docs,
all 25 JSON/five text/four contract hashes and all seven retained capture hashes;
selected/compiled evidence, no private imports and preservation of old fixtures
were confirmed. Security boundaries and product-owned approval remain sound.
Fix P14-W1 under existing local authority; P14-W2 stays blocked on actual human
authority/evidence. The next complete authoritative review uses iteration **2**
after corrections and applicable verification, never a reset to iteration 1.

## Execution evidence

The following paragraphs retain per-phase evidence and counters at their original observations. The live state and authoritative counter above govern current readiness.

Captured source: `refs/heads/main` at `e71c79f98bff9eb95ce139ef6289cd7f2d3f769b`; initially clean. Protocol normalized SHA256 `035d86ec7a288b99f9baece26a1e3fba4a4a385925b72e3718baa23c3c09bd56` matches its registry; all four participating addenda are absent. The invoking approved goal supplies local implementation/test/task-commit authority only; no push, acquisition, live operation or milestone closure is inferred.

Focused P14.1: `go test ./authoringcard -count=1`, retained Go1.26.6, `GOWORK=off GOPROXY=off GOSUMDB=off GOAUTH=off GOTOOLCHAIN=local`, declared ordinary source cache in place and approved private temporary root; exit 0. `gofmt` and `git diff --check` passed. Expensive/protected attempts: 0; whole review: 0/10.

Focused P14.2: the same offline tool/cache policy, `go test ./authoringcard -count=1` and `go vet ./authoringcard`, exit 0. Includes 22 typed-card refusals, six strict-JSON refusals and repeated full projections preserving exact numeric/identity/binding data and escaping multiline/control input. No external effect, new dependency, acquisition or expensive/protected attempt.

P14.3: `go test ./authoringcard ./tests/publicapi ./packagev3 ./approval ./handoff ./wire -count=1` ran all independent cases under the same offline policy. authoringcard (0.144s), packagev3 (20.668s), approval (0.035s), handoff (0.007s) and wire (0.097s) passed; batch exit 1 solely from TestLegacySourceClosureRemainsRemoved in tests/publicapi (0.884s). Its failure is outside P14-owned source: the pre-existing internal directory (mtime 2026-10-25 04:30 UTC) and ignored Oct-1 M92 source files above. They are neither tracked nor P14-created and remain untouched. No full make check or repeat of that failing guard was attempted. The independent frozen public API/import selection `go test ./tests/publicapi -run '^Test(Frozen|PublicTrustImport)' -count=1` passed (0.150s). Final new authoringcard suite with exact-cap/cap+1/depth cases passed (0.078s); vet, format and diff checks passed. One failed non-expensive compatibility invocation; expensive/protected attempts 0; whole review 0/10. No publication/acquisition or consumer-adoption claim.

P14.4 clean-source qualification: detached immutable tracked-source worktree `/home/peter/Workspace/openudon.goal/p14-qualification` at exact `9e1a8c736ad9793aba613d26d1e8cd577c5bc267`; clean before/after, no retired internal/command/debug source paths. The primary main checkout remains documentation/status-dirty and retains its archives. Changed-input `make public-boundary` passed (0.235s), with retained Go1.26.6 and the same offline flags/cache, private `/tmp/k13-8f7f` TMPDIR/GOTMPDIR and process umask 077. Full owner `make check` attempt 1 passed (26.069s) within the 20-minute hard cap; full-check attempts 1, failed 0. Selected/compiled module proof has no replacements/private imports; the owner module remains explicitly local `(devel)` source, never published adoption. [Local qualification](../../docs/p14-local-qualification.json) retains source/schema/corpus hashes, full make output, actual BuildInfo, selected identities and private-capture hashes. A cheap metadata parser was corrected for the local root's absent Sum; the same compiled artifacts were reused without a verification retry. No protected action or unchanged retry. Whole review remains 0/10.

Documentation-only readiness delta: README public-package navigation and docs/public-trust-api.md now link the additive authoringcard data surface, preserving historical trust/package/approval statements. These uncommitted primary-tree documentation changes are hash-bound separately from tested source 9e1; production/schema/test inputs are unchanged, so the passing full check is reused. Fresh read-only P14.4 evidence review and narrow documentation follow-up passed, preserving publication denial and counter 0. No P14.4 task commit, remote action, consumer dispatch or cleanup was performed; the coordinator owns remaining publication/review/closure and evidence retention.

P14-W1 correction: the focused new Unicode regression failed on the old owned helper (48 refusal subcases; exit 1, 0.497s), then passed with the local predecode escape check (exit 0, Go test 0.021s). Full authoringcard tests including the extended 33-case/nine-text draft corpus pass (0.078s), with focused vet/format/diff checks. Valid surrogate pairs, intentional literal/escaped U+FFFD and literal backslash-u remain supported; raw literal keys/values share strict rejection. [Correction lineage](../../docs/p14-unicode-correction.json) preserves red/green hashes and prior 9e1 qualification unchanged. Fresh DEFAULT gpt-6.1-sol/xhigh fix review found no P1/P2; its P3 command-copy quoting issue was corrected and independently confirmed with quoted display text and exact argv. Corrected-source qualification passed below; authoritative counter remains 1/10, parent iteration 2 is pending and P14-W2 remains forbidden/unperformed.

Corrected-source qualification input: local source-fix commit `e3bd5b07b1509e831be834f0e37aaba572f44975`, exact clean detached `/home/peter/Workspace/openudon.goal/p14-unicode-qualification`. Existing worktrees and absent/nonsymlink destination were inspected before creation; no clone/cache copy or old-worktree change. Public-boundary preflight passed (0.188s). Full owner check attempt 2 passed (24.261s) within the unchanged 20-minute hard cap; overall full checks attempted/completed 2 PASS, failed 0. The compiled authoringcard BuildID changed, binding the correction; actual 375-package/374-export graph and 35-module BuildInfo have the same 91 selected/34 external compiled identities as the prior proof, with no replacements/private imports. Old metadata is reused only after exact lock/schema/module identity parity checks; source/corpus/new binary identities remain distinct. The primary checkout has only correction evidence/status updates plus the preserved parent-owned milestone delta; its dirty state is not the frozen verification input. Both immutable qualification worktrees, prior record/seven captures and new private captures remain retained. No whole iteration 2, publication, external action or cleanup occurred.

## Scoped continuation approval — 2026-10-25

The invoking human “I approve” approved both exact reviewed scopes identified
in the [approval record](../../../kinet/docs/stage13-scoped-approval-20261025.json).
The coordinator may create only the named frozen review branch, acquire only
the two exact modules after independent publication verification, and qualify
the named ordinary consumer fixtures. Main/other pushes, deployment and all
other acquisitions remain denied. Publication/consumption are still unperformed;
the row stays suspended while Kinet:M59 owns serial execution. Review counter1,
prior findings and all consumed checks remain; no acceptance is claimed.
