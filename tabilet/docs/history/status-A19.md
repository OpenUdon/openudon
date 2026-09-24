# Retired milestone A19 - A19 Private Browser-Profile Candidate Lifecycle

**Milestone.** A19
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-A19.md
**Source status SHA-256.** 09fbb4c4009caf244403af17808d7f6dd80c03a55e10f12bd8fa4f8cdd47944b
**Source milestone snapshot.** tabilet/docs/history/milestone-before-legacy-retirement.md.txt
**Snapshot SHA-256.** 26884eeda9af4ded30e6d545a33dca84c401d2320c22355fe4fb0336d5dcac71
**Evidence.** 71a4f78afbcf2180fc478ffa89c53544c9160648
**Worktree.** includes uncommitted changes
**Review.** not established
**Review iterations.** not recorded
**Verification.** Original status bytes and full earlier milestone bytes preserved by SHA-256; no fresh acceptance claim.
**Consolidated into.** Current milestone dashboard and maintained memory-bank guidance; full earlier text remains in the frozen snapshot.

## Status record

~~~~~~~~~~~~~~~~~~~~markdown
# A19 Private Browser-Profile Candidate Lifecycle

| Item | State | Notes |
| --- | --- | --- |
| A19.1 Private transaction candidate adoption | `[+]` | Complete at OpenUdon `b5fbc0b6c49ca4a4aeb746afd592f51942b81249`. The published A08 pseudo-version is pinned; the hidden registration worker shares the stabilized mode-0500 re-exec, minimal environment, validated Browsertools Linux sandbox selector, stdout drain, fixed deadline, and full process-tree boundary. An anchored mode-0700 inbox admits exactly one new mode-0600 result only after clean exit, caps it at 256 KiB, rejects symlink/root/file replacement and partial/ambiguous/tampered/stale output, independently rebuilds exact BRP/review/digest/provenance/network/generation facts, binds complete human-confirmed symbolic slots, and returns only defensive canonical source/review bytes plus the session-free candidate transaction. Focused, race, vet, full `GOWORK=off go test ./...`, docs, boundary, and diff gates pass. Task review found one generation/bounds/observation provenance gap; the same commit fixes it and the re-review is clean. |
| A19.2 Virtual browser source discovery | `[+]` | Complete at OpenUdon `3c1a5e7d7b98bcba09b1849d4b41c1429c5c0fc5`. Candidate/review/source digests, canonical producer encoding, supported schemas, earliest expiry, exact origin union, selected BAP/BRP flow, complete symbolic slot coverage, and BAP-provides/BCP-requires session dependencies are revalidated into deterministic `virtual-browser://` plans. Engine snapshots expose only value-free summaries behind an optimistic catalog generation; selection closes dependencies, catalog replacement rechecks selected identity, physical target collisions and stale generations fail closed, API-first order is preserved, and canonical source bytes stay in memory until ordinary approval. Focused, actual-producer compatibility, replacement/collision/traversal, race, vet, full `GOWORK=off go test -p 1 ./...`, docs, boundary, and diff gates pass. Task review found one selected-flow identity gap; the same commit digest-binds flow provenance and the re-review is clean. |
| A19.3 BAP and BCP session composition | `[+]` | Complete at OpenUdon `21f59f22cfd1f102893b91ef542fbe3c44265e41`. Authenticated-authoring now independently composes canonical BAP/BCP source and review bytes into one immutable candidate with one exact flow, earliest expiry, exact origin union, compatible shared contexts, complete symbolic slot bindings, login-state requirement, and an execution-local symbolic session. Explicit acceptance uses the M77 reviewed transition; virtual lowering dependency-closes BCP over BAP and preserves the exact session/bindings through workflow, package inventory, and step-scoped authentication approval derivation without retaining runtime state. Mismatch/ambiguity/non-retention, focused, race, vet, full serial `GOWORK=off go test -p 1 ./...`, docs, boundary, and diff gates pass. Task review found that interactive authoring could replace transaction session/binding names; the same commit carries the exact contract in operation metadata and blocks drift in readiness, and re-review is clean. |
| A19.4 BRP no-session adoption | `[+]` | Complete at OpenUdon `f7d7163b743ef3f635f029bbdbd0c8d0216e0e2e`. Explicitly reviewed BRP transactions expose only their selected flow and lower to standalone `browser_registration` intent with exact symbolic bindings, no browser session or generic operation, a bounded timeout, fixed duplicate/ambiguity/cleanup policy, inert submit and human-checkpoint source material, and a fresh step-scoped authoring confirmation. Ordinary approval independently revalidates and atomically proposes the canonical profile, adjacent review bundle, and derived package inventory; package synthesis proves session-free `uws.browser-registration-call.1.0` lowering, while non-dry execution still rejects before executor invocation. Focused producer/engine/artifact/package/runtime tests, race, vet, full serial `GOWORK=off go test -p 1 ./...`, strict docs, standalone build, docs-memory, boundary, repository, and diff gates pass. Task review iteration 1 found lost in-memory review bytes across clones, missing repair-answer application, and incomplete resumed registration posture; fixes preserve review material, deterministic repair/reapproval, and symbolic route/credential recovery. Iteration 2 found that generic discovery could accept an unreviewed BRP; it now requires the reviewed state with negative coverage. Iteration 3 is clean with no unresolved P1/P2 finding. |
| A19.5 Recovery, drift, and bounded review | `[+]` | Complete at OpenUdon `1fefb8e7672e542a6f6988f5e3cf6a09632c898b`. Exact candidate rediscovery rehydrates selected source/review bytes only in memory and preserves source-scoped approvals; missing, stale, or identity-changed candidates block resume. Deselecting or replacing a virtual source clears affected BAP, BCP, and BRP authoring approvals before deterministic repair and reapproval. Registration cancellation produces no candidate or private result. Existing adversarial coverage proves changed result/root/file identity, partial/ambiguous private writes, stale observations, generation conflicts, workspace drift, and fixed lifecycle transitions. Browsertools and UWS module pins exactly match published heads with no module replacement. Focused and adversarial tests, race, vet, full serial `GOWORK=off go test -p 1 ./...`, standalone build, strict docs, docs-memory, boundary, repository, and diff gates pass. Milestone review iteration 1 found approvals surviving source removal/replacement; source-identity invalidation fixes it. Iteration 2 found exact resume did not rehydrate pre-intent selected bytes; refresh now merges the freshness-validated in-memory plan. Iteration 3 is clean with no unresolved P1/P2 finding, and P05, A20, E10, and W01 are reconciled below. |

Dependencies: OpenUdon M77 and published Browsertools
`39e32c1d6f601561cc5c13ec85201815ce85ab9b`
(`v0.0.0-20260825225202-39e32c1d6f60`). This
milestone does not authorize live registration execution or target access.

M77 reconciliation: adoption starts from a strictly decoded
`openudon.browser-profile-transaction.v1` candidate. BAP+BCP order is
authentication then capability and requires one symbolic session; BRP is one
registration candidate, requires complete symbolic bindings, and forbids a
session. Identity/provenance and source/review/result digests stay immutable
through review; A19 must use the transaction validator rather than inventing
parallel states or UWS fields. Stable reads are limited to the v1 256 KiB bound
and reject invalid UTF-8, duplicate names, excessive nesting, unknown fields,
and trailing JSON before private-result adoption.

Browsertools A08 completion reconciliation: the consumer must accept only the
exact separate registration result/session v1 semantics now present at the
published producer head: accepted-observation time, latest-generation
accessibility-name submit binding, closed diagnostics, GET/HEAD accounting,
context-bounded teardown, independently reconstructed anchored private
persistence, and explicit false submit/account/session/runtime claims. The
parent must reuse its stable executable, minimal environment, stdout drain, and
complete process-tree containment around the new worker, including validated
administrator-controlled Linux sandbox selection, rather than embedding
Playwright into the engine or server.

A19 completion reconciliation: P05 must prepare only from the exact
freshness-validated in-memory virtual source generation and preserve the
separate adjacent BRP review in its atomic byte set. A20 must expose exact
rediscovery conflicts and source-scoped reapproval without surfacing private
bytes or paths. E10 must prove exact resume, changed/missing candidate failure,
approval invalidation, cancellation without adoption, and the published
Browsertools/UWS pins. W01 must classify this local OpenUdon completion as
unpublished and consume only its credential-free contract and commit identity.
~~~~~~~~~~~~~~~~~~~~
