# A32 — Legacy closure removal

**Stage:** Kinet STG-12, Phase B. **Owner:** OpenUdon.
**State:** Pending. No row has started. Review 0/10.
**Source baseline:** `477bf1a53591da70c98979a7a351db4fafac0ff9` (clean at planning).
**Coordinator:** [Stage 12 contract](../../../kinet/docs/stage12.md). This
package-local milestone and status own acceptance. Planning was approved on
2026-10-09 (Kinet R65), where the owner chose full legacy closure removal. It
authorizes these planning files only.

**Lineage.**

- [A31](../docs/history/status-A31.md) inventory (`docs/a31-consumer-inventory.json`),
  which retains six targets for the Stage 12 removal gate.
- Kinet `docs/m48-stage12-handoff.md`.

**Parallel planning provenance.** Stage 12 parallel execution review,
2026-10-09; source priority and separate review baseline not supplied. Current
revalidation `477bf1a53591da70c98979a7a351db4fafac0ff9`, including uncommitted planning files. F01 (confirmed local P2) adopts scoped workflow ownership; F02 (confirmed Lower) replaces strict ordering with readiness; F04 (confirmed local P2) requires frozen consumer checks. F05 (confirmed conditional Lower) permits independent retirement/inventory branches while preserving operational triggers and the C11 join.
Evidence: the old Kinet goal/launch rules, owning agent rules, existing pending
dependencies and sibling consumer checks; Udon `go.mod`/`pkg/execute` import no
OpenUdon code. This approved intake changes no row state or review counter.

## Dispatch and lease boundaries

**Depends on.** OpenUdon:P10, Kinet:M53. All required prerequisites must have
accepted closure at exact revisions; sibling producer adoption also needs
independently verified publication. A priority position never supplies authority.

**Downstream impacts.** UWS:C11?, Kinet:M55?.

**Write set.** The owning `openudon/` package's implementation, tests, ordinary
documentation, manifests and qualification outputs only as required by this
milestone's existing scope, plus `tabilet/memory-bank/status-A32.md` in its
assigned worktree. Excludes `AGENTS.md`, `tabilet/GOAL.md`, shared memory-bank
files, other statuses, evolution, stages, history/knowledge, the package audit
database/sidecars, coordination docs and launch input. The coordinator alone applies shared-memory and closure
changes serially; no child writes a sibling repository or user ledger.

**Contracts read.** Immutable exact prerequisite artifacts listed above, the
M51 native-owner-reviewed contract/fixtures when applicable, the assigned
package baseline and frozen shared-memory/consumer snapshots captured at
dispatch. Cross-package checks use read-only exact snapshots or approved
published module inputs, never changing sibling checkouts. Record full source,
artifact and fixture hashes in the later execution brief; contract drift pauses
affected leases for coordinator reconciliation. Existing no-workspace/no-directory
substitution requirements for ordinary published adoption remain in force.

**Parallel-safe.** yes. Eligible only under the explicit Stage 12 lease opt-in, with no dependency path or bidirectional read/write conflict against any running lease.
At most one live milestone per package. All tests use private lease ports,
disposable stores/caches/browser profiles and unique output directories.

## Dependencies and handoff

**Upstream.**

- [OpenUdon:P10](../docs/history/status-P10.md), the public browser subset.
- Accepted [Kinet:M53](../../../kinet/tabilet/memory-bank/status-M53.md), with its
  exact proof that Kinet consumes no OpenUdon CLI, image or internal package.

**Downstream.**

- [UWS:C11?](../../../uws/tabilet/memory-bank/status-C11.md)
- [Kinet:M55?](../../../kinet/tabilet/memory-bank/status-M55.md)

**Frozen consumers.** W8M (`c2f161d`) and Ramen keep their pinned revisions, which
must stay fetchable. No tag deletion or history rewrite.

## Tasks

| Item | State | Notes |
|---|---|---|
| A32.1 — Refresh the consumer inventory | `[ ]` | Regenerate the A31 inventory at HEAD. Confirm the Kinet:M53 retirement proof, P10's public subset and that frozen consumer pins resolve. Record every remaining production, test and command edge. |
| A32.2 — Remove browser authoring and harness packages | `[ ]` | Delete `browserauthor`, `browserauthoring`, `browsercandidate`, `browsercapture`, `browserpackage`, `browserscenario`, `browsersystem`, `browserintegrationeval`, `browsertransactioneval`, `capturequalification`, `browserworkflow` and `browsercheck`, and the browser CLI commands. Exclude whatever P10 made public. |
| A32.3 — Remove the retained set and legacy CLI | `[ ]` | Delete synthesize, workflowintent, elicitor, projectwizard, udonrunner, trustedrunner and packagepipeline. Delete authoringcli/authoringengine/artifactwriter/stepauthoring and eval/smokematrix/releaseevidence. Delete `cmd/udon-runner`, the legacy `openudon` commands, and image/runner publication. Any CLI that remains is backed only by public packages. |
| A32.4 — Remove legacy HCL adapters | `[ ]` | Delete the intent.hcl/workflow.hcl adapters. Prove no UWS HCL-input API (`convert.HCLToJSON`, `UnmarshalHCL`, `hcl.Import`) remains in any OpenUdon package. |
| A32.5 — Qualify and publish | `[ ]` | Public API and wire fixtures unchanged; v2/v3 history readers intact; Kinet worker and Udon consumer builds pass; docs and memory bank consolidated; publication handoff under named authority. |

## Acceptance and verification

**Acceptance.**

- The refreshed inventory shows an empty legacy closure.
- Only public packages, plus any CLI backed solely by public packages, remain.
- Every public API, wire and history reader is unchanged.
- Frozen consumers still resolve their pins.

**Verification.**

- `GOWORK=off go test ./...`, `go vet ./...` and `make check`.
- Public-closure and boundary guard.
- Inventory regeneration.
- Consumer builds.
- `git diff --check`.

## Execution policy

One coordinator owns the integrated ledgers, shared memory and serialized
integration/closure. Serial execution remains the default. Concurrent leases
require this milestone's declared safety, frozen inputs, a complete explicit
Kinet goal request and the Stage 12 agent-rule opt-in. Each lease has one
in-progress row and one assigned milestone; its persisted review count survives
resume/rebase. Commit policy comes from that later request. Source publication,
deployment and live operations retain separate named authority. Planning and
status markers grant none; audit stays disabled.

## Review

Whole-milestone review: 0/10, not started.

**Accepted P10 reconciliation.** Public browser supplement is accepted/retired at whole review4, exact published source261563d (qualifieda7185841). Public packages browsercontract/browserverify/browsertransaction and additive approval/Authority/evidence remain supported; preserve them in eventual removal. This supplies only P10 prerequisite: accepted Kinet:M53 no-consumer proof is still absent, so every A32 row/review remains pending. No source/pin/audit/live authority changes.
