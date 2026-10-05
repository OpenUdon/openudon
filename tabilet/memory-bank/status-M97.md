# Status M97 — Brokered execution handoff

**State:** Execution started, 2026-10-05. M97.1 completed; the remaining three rows are pending.
**Stage:** STG-09 (Kinet coordination label; milestone IDs remain repository-local).
**Specification:** [M97](milestone.md#m97--brokered-execution-handoff).
**Provenance:** User approved the complete Stage 9 proposal with “Implement the plan” on 2026-10-05. This applies planning-file actions only; a later goal request starts code work. Planning baseline `fbda7e9231b8b306fd1ae3ac623e9d70331b3e08`; [Stage 9 contract](../../../kinet/docs/stage9.md) records discovery evidence and all decisions.

## Scope and dependencies

Own reviewed package/approval/configuration/evidence binding and external private-Udon invocation. Add explicit broker-enabled executor configuration and evidence versions, keeping legacy readers/outputs. No private executor module imports. Bind grant-derived per-run approval to package, concrete inputs/constraints, allowed operations/destinations, executor and credential revisions. Allow the production tier only through this explicit approved broker path; never broaden sandbox destination rules. Values remain outside worker environment, artifacts and reports.

**Requires:** Accepted/published Udon M46 source, contract fixtures and exact executor closure are now reconciled below. OpenUdon baseline fbda7e9231b8b306fd1ae3ac623e9d70331b3e08; existing runtime acceptance c2f161d762bc9f2217bbf0c34b00cdef64b0f7d0 remains frozen.

**Accepted Udon handoff:** M46 source `95c5850fd446e06ac6f79d943774db67e417c989`; independently verified source-record publication `58f9fa5cda92d508e9fcb4085157a09949ddfab9` (completed owner record published and independently verified at `71071537890599e98541abe8ca564490660dfd16`); closing review 2 passed. Frozen executor SHA-256 `53bb9e8976f67c6a5880f07248a99cd195e6285ce7a5a68239793dd5dc6eb429`, fourteen-source closure SHA-256 `a286f21a5dcd4180b7ec19bf0630188ddacec26e0532198db14decd0c86b2069`, broker fixture manifest SHA-256 `7289085b14f744504019ae6d607d351f74522ed3f930ecbbfd159200fd2ddb5f`. Private qualification `/var/tmp/udon-m46-broker-qualification-20261005/closure.json`; use its clean exports and exact executor, not a sibling checkout rebuild. The opt-in contract is `udon.http-broker.v1` plus `--http-broker-config`/report v5, bounded unique straight-line OpenAPI HTTP operations with fixed API-key/bearer bindings; no browser, signing, OAuth, data files, repeated/nested plan or direct fallback. Legacy reports/pins remain unchanged. This satisfies the M46 prerequisite only; M97 implementation and acceptance remain pending.

**Consumers:** Kinet M35 uses the exact accepted/published broker handoff and M97 fixtures; W14 binds approval/evidence and M37 qualifies the bundle. Existing authoring/capture consumers retain their own pins; no W8M adoption.

## Tasks

| Item | State | Notes |
|---|---|---|
| M97.1 — Define approval, configuration and evidence contracts | `[+]` | Publish the broker-enabled versioned handoff with run/grant/policy/credential-reference and exact package/input/executor bindings; preserve existing schemas and readers. Distinguish a bounded recurring grant from the concrete per-occurrence approval emitted by its trusted host. Include mismatch and downgrade refusal fixtures. Defined concrete Authority v1, approval v2, executor config v3 and evidence v4 with strict metadata/digest/deadline/inventory validation, four embedded schemas and seven hashed authority fixtures. Positive Go envelopes validate against the schemas; stale input/credential policy and legacy downgrade tests refuse. Full make fast and focused semantic/schema tests passed offline. Runtime wiring remains M97.2; no new profile is qualified or accepted. |
| M97.2 — Pass broker authority without credential values | `[ ]` | Wire the private Unix-socket/capability references through trustedrunner and the external Udon CLI. Preserve production approval checks and sandbox protection; broker mode bypasses environment-value requirements only for declared broker-resolved references. No host sockets, keys or private Udon imports. |
| M97.3 — Qualify producer and consumer fixtures | `[ ]` | Consume actual published M46 fixtures and closure, publish a Kinet-compatible positive/negative corpus and manifest, and test replay, stale package/input, unsupported version, wrong operation and uncertainty handling. Keep original producer provenance separate from runtime adoption. |
| M97.4 — Qualify, review and publish exact handoff | `[ ]` | Run owner runtime/adoption checks against exact M46 and preserve historical browser evidence. Persist a whole-diff pre-publication review; this row owns scoped publication only under separately confirmed normal-push authority. Verify actual origin/main and hand off full accepted/source/publication hashes to Kinet before normal closing review/retirement. |

## Acceptance and verification

Approved package and exact run authority are inseparable from broker configuration and evidence. A symbolic credential binding works without an environment secret in broker mode; legacy environment behavior remains unchanged. Unsupported/missing/mismatched authority is rejected before Udon invocation. Accepted source, fixtures and publication are consumable by Kinet.

make fast for routine edits; go test ./...; go vet ./...; make check; go run ./cmd/openudon check; go run ./cmd/openudon check-apitools-boundary; (cd tabilet && go run ../cmd/openudon check-doc-memory); fixture validation and git diff --check. Run the required affected make smoke and full make qualify for runtime adoption, including three fresh native repeats under owner rules, in disposable exact-source closures. No cache result substitutes for required fresh qualification.

## Execution and authority

One execution owner works serially in the approved cross-package order:

```text
Kinet:M34 -> Udon:M46 -> OpenUdon:M97 -> Kinet:M35 -> Kinet:A14 -> Kinet:W14 -> Kinet:M36 -> Kinet:U12 -> Kinet:M37
```

The later confirmed goal uses Kinet's existing tabilet/GOAL.md as coordinator with COMMIT_POLICY: task; each package's instructions, local ledger, verification and closure remain authoritative. Kinet holds the only launch reference. Original planning approval wrote plans only. The later Stage 9 goal confirmed task commits and scoped fetch/normal publication of reviewed M97 implementation and closure records to git@github.com-tabilet:OpenUdon/openudon.git. No downloads, installs, deployment, live API/model calls or real email is authorized. This milestone's final row includes a separately authorized publication operation: it must be in progress before its launcher, source must pass pre-publication review, and task state never substitutes for authority. Finish the ordinary post-task closing review after publication; carry its persisted counter across interruptions.

Reconcile actual full accepted/source/publication revisions and hashes before starting consumers. Future upstream hashes are **not yet available**; never invent them or substitute current sibling worktrees for frozen qualified inputs. Preserve unrelated changes and frozen evidence. After verification, persisted review and downstream reconciliation pass, perform this repository's normal retirement. Do not reopen historical milestones.

## Persisted review

- Review iteration: **0/10**; not started.
- Findings: not reviewed; no acceptance is claimed.
- Accepted implementation revision: not yet available.
- Verification/build evidence: not yet available.
- Publication evidence: required before downstream adoption; not authorized by this planning action.
- Downstream reconciliation: pending exact upstream/downstream revisions.
