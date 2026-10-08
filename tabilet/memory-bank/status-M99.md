# M99 — Stage 11 public decoder and review contract remediation

**Stage:** STG-11 post-acceptance remediation. **Owner:** OpenUdon.
**State:** Approved planning, 2026-10-08; implementation not started. All 3 task rows are pending.
**Authority:** The user approved the complete reconciliation proposal with “Implement the plan.” This applies planning files only. A separate execution request is required; no code, commit, source-publication, deployment or live-operation authority follows.
**Review source:** stage11-siblings-review.md — Stage 11 code review — sibling packages; OpenUdon section.
**Review baseline/range:** `7cd7fbb837fb87e1ca4abea2a362790b0f434188` → `f7aa5d874bc474888bac1b43d4112c1faf29d499`.
**Revalidation HEAD:** `f7aa5d874bc474888bac1b43d4112c1faf29d499`; clean worktree, no relevant uncommitted code in the evidence. Approved planning changes are not implementation evidence.
**Lineage:** [M98](../docs/history/status-M98.md) and [P09](../docs/history/status-P09.md); [A31](../docs/history/status-A31.md) trust/compatibility boundary remains frozen. Existing acceptance, review counters, statuses and frozen evidence stay preserved.
**Coordinator:** [Stage 11](../../../kinet/docs/stage11.md#post-acceptance-remediation--2026-10-08); package-local specification and status own acceptance.

## Dependencies and handoff

Accepted M98/P09/A31 contracts, accepted and independently published UWS:M09 root/codec and APItools:M83 before exact SDK adoption/qualification. Serial scheduling follows APItools:M83. No private Udon import is added; runtime proof remains supplied through public host adapters.

**Exact successor acceptance/publication/build identities:** unset; record full independently observed revisions and hashes during the later execution. Never substitute local HEAD, directory replacements or prior consumed publication authority.

**Downstream:** Kinet:M49 public author and separate private exec consumers, exact package/source verification, corrected worker closures and successor bundle. Browser/legacy/frozen consumers keep their existing independently fetchable pins.

## Scope and acceptance

Make strict JSON key handling match typed Go record semantics without over-rejecting free-form data, and retain unsupported symbolic security metadata as indeterminate for review-only packaging. Preserve exact valid wire/schema bytes, public/private import boundaries and no-public-v2-build/synthesis contract.

Struct-typed records reject duplicate aliases under Go Unicode simple field folding, including long-s/K cases, at nested typed paths. Free-form maps permit case-distinct keys while still rejecting exact duplicate keys and retaining number, depth, node, Unicode and trailing-value bounds. Build/Assess/Verify preserve unsupported scheme names as original indeterminate review metadata; broker approval/runtime binding stays refused unless its independently supported/addressable policy is proved. Existing valid historical wire encodings, digest order, source verification, public API manifests and read-only v2/v3 history remain compatible. Ordinary public/private consumers and whole milestone review qualify exact published SDK closure.

## Tasks

| Item | State | Notes |
|---|---|---|
| M99.1 — Apply typed Unicode alias checks without rejecting free-form maps | `[ ]` | Make duplicate checking aware of destination struct/map types at nested paths, matching encoding/json Unicode simple field folding for records. Permit id/ID in free-form JSON while rejecting exact duplicates everywhere; preserve bounded decoding, numeric lexemes and trailing/unknown field policy. Test run-evidence v2/v3, package data and protected broker/package boundaries. Sources P3.1/P3.2. |
| M99.2 — Preserve indeterminate symbolic security in review-only packages | `[ ]` | Build/Assess/Verify retain original unsupported scheme symbols without silently renaming or treating unknown security as anonymous. Keep credential/addressability/broker approval requirements closed, refusing execution when unsupported names cannot be independently bound. Preserve valid source/shape security semantics and wire/schema contracts. Source P3.3. |
| M99.3 — Qualify the corrected public SDK and source handoff | `[ ]` | Adopt exact accepted/published UWS:M09 and APItools:M83, run public import/API manifests, unchanged valid wire/digest/schema, forged-source verification and ordinary public/private consumer checks. Keep v2 synthesis/build APIs private and A31 browser closure unchanged. Persist pre-publication/closing review counts, publish only under fresh named authority, and hand off exact SDK sources/sums to Kinet:M49. |

## Active finding provenance

Only approved active findings are recorded here. Source priorities and local severity are separate; task references identify one owner for each required outcome. Unsupported and unscheduled findings remain in the conversational handoff; optional directions belong only in milestone.md.

| Source finding | Source priority | Local severity | Disposition | Current repository evidence | Task owner |
|---|---|---|---|---|---|
| P3.1 | P3 | P2 | confirmed | wire/json.go:95 uses strings.ToLower; offline scope/ſcope probe passes but encoding/json selects the alias | M99.1 |
| P3.2 | P3 | P2 | confirmed | packagev3/build.go and wire.DecodeStrictNumbers reject valid id/ID free-form map; offline probe | M99.1 |
| P3.3 | P3 | P2 | confirmed | packagev3/security.go:44 hard-refuses unsupported symbols even for Build/Assess/Verify review-only paths | M99.2 |

## Verification and compatibility

go test ./...; go vet ./...; owner quality/API-surface/public trust-import/wire/schema identity checks; affected wire/packagev3/runevidence races; Unicode struct alias/exact duplicate/map-key/number/depth fixtures; review-only unsupported security with execution refusal; forged/stale source/shape/assessment/authority regressions; exact accepted UWS/APItools standalone module/public/private consumer reproduction with GOWORK=off GOPROXY=off; git diff --check.

Use retained Go 1.26.6 and exact ordinary modules without ambient workspace substitution. Default verification is offline, credential-free and model-free. No live user ledger, host, provider, account, mail, Cloudflare, registration or consumer adoption operation is included.

Public schemas/wires, published grammar/version bytes, accepted historical qualification and independently retained browser/media/legacy/frozen-consumer pins stay preserved. Corrected derived metadata and new worker/package identities require fresh consumer assessment and explicit authority; historical approvals are never upgraded automatically. Source publication requires a new separately named request and independent resolution before downstream adoption. Planning rows may remain pending on this external prerequisite; none is started here.

## Closing review

**Review iterations:** 0/10.
**Review state:** not started; this is review intake, not a pass of an existing gate.
**Findings/fixes:** no implementation or fix verification claimed.
**Execution owner:** one serial owner across the five ledgers; no row is in progress.
**Commit policy:** The user separately authorized a planning commit on 2026-10-08 with “git commit and then report the index refresh issue in ~/skill-index.md”. This authorizes one commit of the approved planning changes in this owner repository; implementation, publication and deployment remain outside this request. Future task commits follow the separately invoked GOAL/request policy.
**Closure:** persist each started review iteration before reviewing; resume an interrupted pass at the same number. No open P1/P2 may remain at acceptance. Required verification, exact downstream reconciliation and owner-specific consolidation/retirement follow implementation; never reopen completed Stage 11 history.
