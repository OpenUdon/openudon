# A32 — Legacy closure removal

## Current Stage 12 follow-up order

Kinet:M53 is accepted/retired for early local evaluation. The owner-approved
Kinet:M58 test-only findings milestone runs first by priority, then OpenUdon:A32,
then Kinet:M54? → Udon:M54? → UWS:C11? → APItools:M86? → Kinet:M55?. Current
horizon: seven milestones / 29 rows (required two / ten, conditional five / 19).
Use the single [Kinet launch reference](../../../kinet/tabilet/memory-bank/suggested.txt).
M58 adds no blanket A32 acceptance dependency. Hold A32 only when findings show
that removal would break retained behavior or a required public contract; an
owner-approved repair dependency must resolve that blocker first. Other gaps do
not automatically hold A32. A32 rows and review counters are unchanged.
M58's implementation successor remains unnumbered until its findings are
accepted after closure. No new publication, deployment or Phase C authority.
Registration stays denied; audit stays disabled. Earlier dated checkpoints are
historical; this paragraph and the A32 specification define current readiness.


**Accepted Kinet:M53 early-use handoff — 2026-10-24.** Kinet completed all39 rows, task4/task5 reviews1/1 and fresh whole review1 under the owner-approved early local evaluation standard. [Exact dependency/artifact/limitation handoff](../../../kinet/docs/m53-stage12-handoff.md) and [retired M53](../../../kinet/tabilet/docs/history/status-M53.md) supply the dependency evidence. Selected native and retained default/Phase A/media paths use Kinet-owned workers/images, not the legacy OpenUdon CLI/image/internal producer. Public libraries (including live default801b), private Udon/HCL and P11/P12/P13 contracts remain. Additional physical/observer/forced-crash methods are deferred/unqualified, not A32 prerequisites or passed tests. Preserve unknowns/no replay/registration denial/disabled private input. A32 rows remain pending/review0; no publication, deployment or Phase C authority is added.


**P13 preservation prerequisite — 2026-10-17.** A32 additionally depends on
accepted/published OpenUdon:P13 and accepted Kinet:M53. Preserve the public
requested-authority/observed-fallback-rights contract and its deterministic
expiry/evidence regressions through A32.1 inventory and A32.5 verification.
P13 is now accepted/retired at whole review1 with exact source5f210465 and complete ordinary-module proof. Preserve its public regression contracts; A32 rows/review remain pending on accepted Kinet:M53. Normal P13 closure publication is verified before consumer adoption.

**P12 public preservation gate — 2026-10-13.** Depend on accepted/published
OpenUdon:P12 and Kinet:M53's exact no-legacy-consumer proof. A32.1 inventories
P12's public browser authentication predicate/plan path; A32.2/A32.5 preserve
the authored bounded-auth correction and additive public/native compatibility
regressions, existing complete hash/wire semantics, P11 outputs and readable
untimed history. Retiring legacy authoring must not remove or widen the public
path. Frozen consumer pins, all existing row text/markers and review0 remain
unchanged. P12 owns the repair; A32 starts no duplicate row. Audit disabled.

[Accepted P12 specification/status](../docs/history/status-P12.md).

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

## Status rules

See [`milestone.md`](milestone.md) for milestone definitions and the status ID
pattern that names this file.

Status markers:

| Symbol | Suggested Status | Interpretation |
|---|---|---|
| `[ ]` | Pending | Item is actionable when its dependencies pass. |
| `[+]` | Completed | Item finished or done. |
| `[~]` | In Progress | Item is the one general row currently being worked. |
| `[!]` | Blocked | A current unresolved blocker still prevents progress. |
| `[X]` | Cancelled | The approved item was withdrawn; any later replacement needs fresh planning. Cancellation does not prove delivery. |
| `[-]` | Closed Historical | A consumed failed attempt or superseded row retained for audit; it is never retried and does not block its accepted successor. |

Write markers with backticks, exactly as in the table above: `` `[ ]` ``, not
`[ ]`.

Across the active ledger, zero or one general row may be `[~]`. Resume it before
selecting another row. An operational launcher additionally requires its exact
authorized operation row to be `[~]` before invocation; the marker records the
selected operation but does not grant missing external-mutation authority.

Read this milestone's optional `AUTHORIZATION_REQUIREMENTS` in
[`milestone.md`](milestone.md#authorization-requirements); do not duplicate its
declaration here. Check protected actions against the current human-approved
request and effective grants before acting. Notes may record a grant ID,
performed-action evidence, or a missing-authority blocker; they never prove
approval. Required unperformed actions prevent closure. No new task marker is
introduced, and absent authorization metadata grants no additional authority.

With human-approved `CORRECTIVE_PLANNING: auto`, the owner may prepare and
check necessary repair plans within the original delivery outcome, existing
grants and explicit limits. Preserve this row's identity and interrupted work;
move the interrupted `[~]` row to `[!]` before activating a repair row. Record
prerequisite IDs, standing approval source and resumption condition. Keep zero
or one `[~]` row. Once every prerequisite is verified and accepted and all
blockers clear, restore this row from `[!]` to `[~]` under the same approval.
Recover this chain on resume; do not select unrelated work. Preserve partial
implementation; suspension bookkeeping may join the scoped repair checkpoint,
but unfinished implementation cannot. Pause if overlapping changes cannot be
separated safely. Explicit project restrictions still govern.
Preserve review counts and frozen history. Children report needs
to the owner; a marker never grants planning or action authority.

Each table row is a commit unit: after changing its state, verify the change,
update the memory bank/docs, and make a scoped `git commit` before starting the
next row, except the scoped repair suspension bookkeeping above. A `[-]` row's
notes must record the consumed attempt or supersession
and identify its accepted successor. If two rows are inseparable, redefine them
as one row before starting rather than closing several rows in one commit.
During a goal run, the resolved `COMMIT_POLICY` governs instead; `none` means
no commits even when a requirement or grant mentions `git.commit`.

Keep this file a current-state record, not a log. Replace a superseded checkpoint
instead of appending dated narrative, and link per-attempt detail in evidence
files. A row whose acceptance depends on an expensive check records its attempt
count here under the [verification-attempt gate](milestone.md#verification-attempt-gate).

After the milestone's review, consolidation, and downstream reconciliation
pass, follow [the retirement procedure](milestone.md#long-term-memory-and-retirement).
Preserve this complete document and the full milestone specification in its
retired record; earlier rows, notes, and original path context must survive.
Retirement follows the governing commit policy and never changes the task
markers or permits reuse of this milestone ID.

## Dispatch and lease boundaries

**Depends on.** OpenUdon:P10, OpenUdon:P11, OpenUdon:P12, OpenUdon:P13, Kinet:M53. All required prerequisites must have
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
- [OpenUdon:P12](../docs/history/status-P12.md), the required public authentication correction, after exact accepted/published closure; preserve [P11](../docs/history/status-P11.md).
- Accepted [Kinet:M53](../../../kinet/tabilet/docs/history/status-M53.md), with its
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
| A32.2 — Remove browser authoring and harness packages | `[ ]` | Delete `browserauthor`, `browserauthoring`, `browsercandidate`, `browsercapture`, `browserpackage`, `browserscenario`, `browsersystem`, `browserintegrationeval`, `browsertransactioneval`, `capturequalification`, `browserworkflow` and `browsercheck`, and the browser CLI commands. Preserve the P10 public subset including accepted P11 named-output resolution and historical-proof refusal. |
| A32.3 — Remove the retained set and legacy CLI | `[ ]` | Delete synthesize, workflowintent, elicitor, projectwizard, udonrunner, trustedrunner and packagepipeline. Delete authoringcli/authoringengine/artifactwriter/stepauthoring and eval/smokematrix/releaseevidence. Delete `cmd/udon-runner`, the legacy `openudon` commands, and image/runner publication. Any CLI that remains is backed only by public packages. |
| A32.4 — Remove legacy HCL adapters | `[ ]` | Delete the intent.hcl/workflow.hcl adapters. Prove no UWS HCL-input API (`convert.HCLToJSON`, `UnmarshalHCL`, `hcl.Import`) remains in any OpenUdon package. |
| A32.5 — Qualify and publish | `[ ]` | Public API and wire fixtures unchanged; P11 output/proof regressions and v2/v3 history readers intact; Kinet worker and Udon consumer builds pass; docs and memory bank consolidated; publication handoff under named authority. |

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

**Accepted P11 preservation handoff — 2026-10-13.** Preserve the current
P10 public subset including [P11](../docs/history/status-P11.md) native output
resolution, deterministic assessment and read-only historical-proof behavior.
A32.1 inventories it and A32.5 retains its regressions/history readers.
Kinet:M53 remains the gate; all A32 rows/review stay pending0/10.

**Accepted P12 handoff.** Four rows and whole review1 pass at published source `54cd50d0b4a688fc1a2a32e679b4e6befc0f5114` with exact ordinary source/H1 proof. Normal closure is independently verified before consumer use. Preserve its public predicates, additive timeout regressions, complete hash/wire semantics and readable untimed history while removing legacy closure. A32 rows and review0 remain pending on accepted Kinet:M53; no implementation begins here. Audit disabled.
