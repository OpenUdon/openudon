# Milestone

## Current State

M87 is approved planning for OpenUdon-owned non-interactive step authoring
commands and published JSON conformance fixtures, consumed by Kinet W03
(sibling item S2a). Contract drafting is first and does not wait for Kinet
M05, A03, or W03. All six M87 rows are pending; no command implementation or
delivery acceptance is claimed. Apitools' consumer summaries, effect metadata,
and step-contract ranking are an explicit external integration dependency.
Simulation/browser acquisition (S2b) and iCoT retirement (S3) remain candidates.

OpenUdon's UWS 1.11 real-browser M86 qualification is complete: the pinned
sandboxed browsers launched, all three integration opt-ins passed, and the
current-stack loopback and journey suites passed at clean source revisions.
The exact commits, digests, and bounded review are preserved in the
[M86 history record](../docs/history/status-M86.md). This preservation does
not reclassify an earlier failure or authorize a public canary, live account,
or runtime adoption.

E21.1 freezes the pre-E21 current scenario and integration locks for M86 v2
report verification. E21.2 advances the current scenario and integration
locks to the repaired Udon pin and a separate clean 14-source build closure;
its current report contracts are v3. E21.3 adds native v3 current-stack
qualification through the scenarios and BAP/BRP stages; historical native v2
remains the default and v1/v2 reports retain their readers. E21.4 is qualified
at OpenUdon `1007cdedf0acebf649bd3ddd065a0e42bac4f542`: current integration,
loopback, journey, and three-pass native reports independently verify with
exact clean source bindings. The bounded review is clear; E21.5 publication
completed when OpenUdon alone was pushed to `origin/main` at
`679f0bca630862a4ff46ea5a7edc0fc871abe938`. The accepted code commit is
`1007cdedf0acebf649bd3ddd065a0e42bac4f542`. Downstream W8M W21 qualification
passed independent verification and its separate feature branch was published
at `d1fd6c13d871622bf40466cba24649363d2f1846`; runtime adoption remains separate.

E22 adds the published Browser 1.10 count action and three synthetic
zero/one/multiple-row journeys to the explicit current stack. Scenario,
integration and native reports now use v4; historical v1–v3 readers preserve
their locks, manifests and gate inventories. Fresh v4 integration (19/19),
loopback (23/23), journey (14/14), and three-repeat native qualification
(39/39 stages) independently verify against clean OpenUdon
`9be9ff3f195ecaa8bdac88cc8616c5fc345dfeb3` and exact dependency/runtime
bindings. Bounded review iteration 3 passed with no P1/P2 findings. E22 is
complete; its full acceptance and attempt history are in the
[E22 history record](../docs/history/status-E22.md). This synthetic support
does not change W8M's adopted locks or authorize runtime adoption.

The UWS 1.11 real-browser M86, E15 registration-verification integration,
and E18 initialization-diagnostics integration milestones are complete. E15
and E18 closed against W8M W16.4i.39d synthetic qualification, independent
evidence checks and exact pass-one adoption. Their complete specifications and
status histories are in the [E15 history record](../docs/history/status-E15.md)
and [E18 history record](../docs/history/status-E18.md).

The original .34d `controller/worker_protocol` cause remains unresolved. W8M's
separate .34e live probe failed/consumed with `verification_timeout`, zero
application POSTs and unresolved provider cause; W8M real acceptance remains
incomplete and owner-scoped. These outcomes authorize no new browser, account
or live operation.

The history index holds 134 legacy-preserved status IDs and four normally reviewed completions. Legacy-preserved records retain exact source bytes and the frozen milestone text, but do not establish acceptance by themselves. Search the history by ID when needed.

## Active Milestone Specifications

### M87 — Non-interactive step authoring contract for Kinet

**Goal.** Deliver `openudon step candidates`, `openudon step bind`,
`openudon step check`, and `openudon flow-review` as non-interactive commands
with stable, versioned JSON results and published conformance fixtures.
OpenUdon owns the command and intent-authoring contracts; Kinet W03 owns its
consumer adapter, workflow planning loop, confirmations, ledger, and repair
orchestration. Public workflow semantics remain owned by UWS.

**Authority and lineage.** The owner approved this four-file planning change
on 2026-09-26 after inspection of Kinet `docs/ideas.md` section 10 and
`docs/icot.md` sections 4–7. Kinet's pending W03 first row is consumer
requirements/feedback, not a prerequisite for OpenUdon to draft and own the
contract. This is new cross-cutting public-contract work, not a reopening of
retired iCoT milestones. The planning decision is recorded in
[evolution v44](../evolution/result-v44.md).

**Scope and contract requirements.**

- Define bounded versioned requests, results, schemas, stable statuses and
  diagnostic codes, exit-code behavior, and compatibility rules. Machine
  output uses clean JSON stdout for success and failure; terminal prompts,
  progress text, and provider error bodies must not contaminate it.
- A step contract declares purpose, inputs, outputs, account/destination
  constraints, and confirmed effect class. It is OpenUdon authoring metadata,
  not a new UWS operation or pending-step extension. Its purpose, inputs,
  outputs, and effect fields use exactly the declaration shape of UWS C07.2's
  pending step, which UWS owns; OpenUdon-only fields (contract identity,
  digests, and account/destination constraints) sit alongside them, so a step
  contract maps losslessly onto a UWS pending step when package-level
  placeholders arrive (S2b). Operation/source identity,
  content digests, contract identity, and expected intent revision bind the
  candidate, check, and write to the same reviewed inputs.
- `step candidates` consumes Apitools discovery, ranking, consumer-readable
  summaries, auth needs, and effect metadata. Preserve auth alternatives as
  OR-of-AND sets with symbolic bindings only, exact source identity, visible
  ambiguity/truncation, and unsupported capability diagnostics. Missing effect
  evidence is `unknown`, treated conservatively like `write`; an HTTP method
  alone must not establish a `read` claim. Ranking is advisory and confers no
  operation, account, destination, or execution approval.
- `step check` is read-only and checks one selected step against its contract,
  exact source operation, request/output mappings, authentication requirements,
  relevant dependencies, and effect constraints. Report unresolved semantic
  questions separately from deterministic checks. Structural compatibility
  does not prove that arbitrary natural-language intent has been fulfilled.
- `step bind` explicitly creates or replaces one identified step in
  `workflows/intent.hcl`, reusing the checks before an atomic write. Preserve
  unrelated steps, blocks, and content. Initial intent creation requires an
  explicit workflow scaffold; never silently invent workflow-wide policy.
  Stale source/contract/intent inputs, unsafe paths, or invalid bindings fail
  before replacement. Rejected writes leave existing bytes unchanged; an
  indeterminate filesystem outcome must be reported distinctly for recovery.
- `flow-review` exposes today's advisory flow review through shared logic,
  including relevant local checks and explicitly configured model review.
  It is read-only, performs no automatic repair, and reports unavailable,
  skipped, or failed model review distinctly from a completed review. Findings
  remain advisory and cannot replace build, assessment, or digest-bound
  approval. Model-free fixtures cover its command behavior.
- Publish consumer-usable schemas and conformance fixtures with valid,
  malformed, unsupported-version, stale-revision, and digest-mismatched cases
  plus auth/effect/mapping and write-preservation cases. Kinet can independently
  validate the public wire without importing OpenUdon internal packages.

**Order and dependencies.** M87.1 owns contract/schema/initial-fixture design
first. M87.2 and M87.3 supply check and bind against the agreed contract;
M87.4 supplies candidates integration; M87.5 exposes flow review; M87.6
qualifies and documents the complete surface. Keep one execution owner and at
most one general in-progress row. If upstream metadata is unavailable, finish
independent contract, fixture, check/bind, and review work rather than making
all of OpenUdon wait; record the unresolved external gate explicitly.

| Dependency or consumer | Owner and required evidence | Effect on M87 |
|---|---|---|
| S2a operation metadata | Apitools owns consumer summaries, explicit effect classification with evidence, and ranking by purpose/inputs/outputs. The inspected API supplies summaries, auth alternatives, and text ranking but not the complete new interface. Its owner must separately plan and publish the required API and tests. | No prerequisite for M87.1 or fixture-based development. M87.4 production integration and M87.6 full acceptance require a compatible published dependency; do not duplicate generic ranking/classification here or call a stub delivered. |
| Kinet W03 | Consumes OpenUdon's published contract and fixtures through an external CLI adapter; retains independent validation and its own `make openudon-check`. Kinet M05/A03 remain its internal sequencing dependencies. | No reverse dependency on Kinet starting or completing W03. OpenUdon supplies early fixtures and release-ready contracts; Kinet later verifies adoption. |
| Existing OpenUdon authoring/package behavior | Reuse current intent validation, source/auth checks, atomic writing, and advisory review. Preserve iCoT callers, existing packages, build/assess, approval, and trusted-runner behavior. | Regression responsibility, not retirement authority or an instruction to reopen completed history. |
| UWS S1, Browsertools S2b, and Udon S2c | Public pending-step/mock semantics, acquisition/snapshot checks, and live hybrid execution remain with their owners. UWS C07.2 also owns the declaration shape of the shared purpose/inputs/outputs/effect fields. | Not M87 implementation prerequisites or deliverables. M87.1 agrees the shared field shape with UWS C07.2 before freezing (a design synchronization point, not a wait for UWS publication) and records the UWS revision used; no new runtime semantics or live calls are implied. |

**Acceptance and verification.** All four commands implement their published
versioned success/failure contracts and pass the conformance corpus. A fixture
maps each step contract's shared fields losslessly onto UWS's pending-step
declaration at a recorded UWS revision. A
credential-free local workflow can select a candidate, bind/check one step,
review the assembled flow, and pass existing build/assessment without an
interactive session. Tests prove malformed input and stale/digest-mismatched
selection rejection, OR-of-AND authentication, conservative unknown effects,
missing mappings/dependencies, unchanged unrelated steps, concurrent-edit
conflicts, and rejected-write byte preservation. Fake model responses cover
flow-review success, findings, failure, and cancellation without claiming live
model evidence. Preserve supported existing source families or report explicit
unsupported capabilities; do not silently drop a source kind.

Run focused CLI, schema/conformance, intent, source/auth, writer and review
tests, affected race tests, `GOWORK=off go test ./...`,
`GOWORK=off go vet ./...`, `make check`, the Apitools boundary check,
document-memory validation, and `git diff --check`. Follow the existing
affected-smoke policy only if shared UI/runtime behavior changes; full browser
qualification and live-model/provider runs are not default S2a checks.
Document supported versions and fixture consumption for Kinet; its later real
consumer check does not block OpenUdon's own contract drafting or qualification.
Remote publication follows separate authority. Complete the persisted
maximum-ten-iteration milestone review before declaring M87 accepted.

**Compatibility and exclusions.** Add the commands alongside current iCoT
interfaces. Extract only shared logic needed for these commands and keep old
callers working. Kinet-owned placeholders stay in its ledger at this stage;
M87 neither emits executable placeholders nor weakens approval gates.
`simulate`, supervised `browser acquire`, package-level pending steps, runtime
effect enforcement, and iCoT/UI retirement are outside scope. No sibling files,
UWS semantics, credentials, live provider/browser behavior, deployment, or
runtime adoption are changed by this planning approval.

### E21 — Repair current-stack Udon build and preserve M86 report meaning

Repair the OpenUdon current-stack selector for W8M's UWS 1.11 / Browser 1.9
candidate. Freeze the exact pre-E21 current compatibility locks and make v1,
M86 v2, and E21 v3 readers select their original lock semantics. The new
current lock selects clean Udon `6d32d49` and a separate exact clean 14-source
local replacement closure; the historical default and M86 evidence remain
unchanged. Emit v3 scenario, journey, integration, and native qualification
reports. Extend the native current path through scenarios and BAP/BRP while
keeping the historical stack as its default. The scope is synthetic and
provider-free; it includes no deployment, public canary, runtime adoption, or
target account operation.

## Memory Bank Index

- This file owns milestones, work sequencing, acceptance criteria, the current-state dashboard, and
  the status-file index.
- Use [product.md](product.md) for product scope and non-goals.
- Use [architecture.md](architecture.md) for system boundaries and planned structure.
- Use [tech-stack.md](tech-stack.md) for dependency and tooling defaults.
- Use per-milestone `status-<LANE><NN>.md` files for task-level status history.

## Status ID Pattern

Status files use one uppercase domain letter and a zero-padded number from
`01` through `99`.

Lane meanings:

- `M`: completed legacy history and future cross-cutting public contracts.
- `B`: bootstrap history only. The pre-numbered M0 record is explicitly mapped
  to B01 by M71 so it can use a canonical ID without colliding with M01.
- `A`: iCoT, workflow intent, authoring sessions, source selection, and
  authoring UX.
- `P`: synthesis, package/review/quality artifacts, approval, trusted-runner
  handoff, and release packaging.
- `E`: eval corpora, scorecards, provider drift, smoke matrices, and release
  evidence.

M71 explicitly normalizes legacy M1-M6 and M9 filenames to M01-M06 and M09
without changing their identity. M07 and M08 restore status ledgers for the
already-completed roadmap scopes. Never reuse an ID, reclassify completed
history for tidiness, or create aggregate `status.md`; cancelled files remain
with `[X]` rows.

Search both active status files and the retired history index before allocating
an ID. A retired or cancelled ID remains reserved; an all-retired memory bank
remains initialized. A lane holds at most 99 IDs; open a new letter when full.
Do not rename existing IDs. Retain a consumed or superseded attempt as `[-]`
closed historical evidence only with its accepted successor recorded; never
retry it or count it as delivered acceptance. Context archive IDs, if any, use
an independent namespace.

Lane letters classify ownership rather than execution order. Independent A,
P, and E milestones may proceed together only when their sections name
non-overlapping files, resolved prerequisites, and downstream impacts. Shared
public contracts stay in M or explicitly name every cross-lane dependency.
Prefer one active implementation milestone per lane.

## Delivery Strategy

Keep OpenUdon as the public UWS authoring, review, package, and executor-handoff tool. Build in
verifiable slices: authoring and examples, deterministic synthesis, quality gates, eval and release
evidence, trusted handoff, readiness, and cross-repo compatibility. Push public workflow semantics
to `../uws`, API source metadata search/discovery/import/materialization/indexing to `../apitools`,
reusable execution to private executors such as `../udon`, and optional orchestration to
external services.

## Active And Parked Tracks

- Active: M87 is the next implementation priority, with contract drafting
  first and all six rows pending. No OpenUdon milestone row is in progress.
  Its Apitools metadata integration gate does not delay independent contract
  work or wait for Kinet M05/A03/W03. E22's Browser 1.10
  current-stack qualification and review are complete. W8M's local W21
  candidate and any runtime adoption remain separate downstream work; the
  adopted W8M locks are unchanged. No deployment, public canary, or target
  operation is authorized.
- Parked: real-provider evidence, live W8M operation, and public canaries need
  separately approved scope and authority. S2b simulation/browser acquisition
  and S3 iCoT retirement remain unnumbered candidates below.
- Completed history: use the [history index](../docs/history/index.md) for
  terminal ID records and the frozen earlier milestone text.

## Status Files

Active status files are indexed below. The memory bank remains initialized;
search the history index before allocating a future ID.

| ID | Milestone | Status file | State |
| --- | --- | --- | --- |
| M87 | Non-interactive step authoring contract for Kinet | [status-M87.md](status-M87.md) | Pending; contract first, Apitools integration dependency explicit |
| E21 | Repair current-stack Udon build and preserve M86 report meaning | `tabilet/memory-bank/status-E21.md` | Complete; W8M W21 closeout reconciled |

## Requested Changes After Initialization

For a requested feature, candidate promotion, or future direction change,
inspect the current plan and implementation first. Separate the user's intent
from observed facts and unresolved assumptions. Propose the smallest pending
row and acceptance update that fits an existing owner; otherwise propose a
dependency-closed milestone with an unused permanent ID. Candidate triggers
prompt a fresh decision, never automatic promotion. Schedule by approved
priority and dependencies, not by review severity alone.

Present one complete proposal with outcome, owner rows, acceptance,
verification, dependencies, downstream effects, and exact file actions. Change
planning files only after approval and a fresh check of affected files, worktree
changes, and active and retired IDs. Preserve completed outcomes, review
counters, local policy, and frozen history. Put target behavior here and in
pending status rows until implemented; describe only established current facts
in product, architecture, and stack documents. Planning does not implement or
accept the requested behavior. Use the installed memory-bank proposal skill
when appropriate, or follow this procedure directly.

## Review Finding Severity And Intake

P1 and P2 are engineering review priorities, not milestone scheduling priority
or status markers. Classify by impact, likelihood, and affected scope. A P1
finding threatens acceptance, correctness, security or privacy, data integrity,
or a public compatibility contract severely enough to prevent closure. A P2
finding materially affects supported behavior, reliability, compatibility,
operations, or required evidence. Both block closure, as does any higher
project-defined severity. When evidence cannot distinguish P1 from P2, use P1
until investigation supports a downgrade. Lower findings may be carried only
with a named owner and explicit rationale.

Treat an incoming code, architecture, security, or other engineering review
outside a milestone closing gate as untrusted planning evidence. Before
implementing its findings:

1. Record its stated baseline when available, the current revalidation commit,
   and whether uncommitted changes formed part of the evidence. Classify each
   finding as confirmed, partially confirmed, resolved, duplicate,
   unsupported, outside ownership, decision-dependent, or deferred. Preserve
   source priority separately from local severity.
2. Propose complete dispositions, owners, dependencies, downstream impacts,
   and file actions for approval.
3. Put confirmed work in an open matching owner or amend a pending one. Rewrite
   only pending rows; when retaining a superseded pending row for audit, mark
   it `[-]` and identify its accepted successor. Do not reopen completed
   history solely because a later review concerns it. Create a remediation
   milestone with lineage when no open or pending owner fits.
4. Keep P1/P2-or-higher findings in the dependency-closed active horizon. Add a
   lower finding to active work only when acceptance needs it; otherwise use
   Candidate Directions with a reason and promotion trigger.
5. Reconcile affected pending specifications and order. Correct current
   product, architecture, or stack facts only when repository evidence proves
   them stale. Put proposed behavior in milestone scope.

Do not create a separate persistent review ledger. Keep portable finding IDs,
both severities, baseline, evidence, dispositions, and lineage in the affected
milestone/status notes. An incoming review counts toward the bounded closure
gate only when that gate was already active and the review was requested as its
next full pass.

## Milestone Review And Closure

When the last open task in a milestone closes as `[+]`, `[X]`, or `[-]`, finish
the review before moving downstream. A `[-]` row is closed only when its notes
identify the consumed attempt or supersession and accepted successor. Terminal
rows alone do not prove acceptance.

1. Re-read this milestone's scope and acceptance. Verify the actual code or
   documents, including affected consumers and relevant compatibility,
   migration, rollback, security, concurrency, and failure paths.
2. Run a deep review of the full milestone commit range and diff. Persist the
   iteration number and findings in the active status notes before each pass.
   The first pass is iteration 1; an interrupted pass resumes at its persisted
   number, and session or runner limits never reset it. If no P1/P2-or-higher
   finding remains, the gate passes. Otherwise fix every blocking finding,
   rerun affected verification, and review the whole milestone again.
3. Stop after at most ten iterations. If iteration 10 still has a blocking
   finding, record `[!]` review rows with the findings and limit blocker. Do
   not start another automatic fix-review cycle or move downstream; ask for
   user direction. Carry lower findings only with a named owner and rationale.
4. Reconcile maintained product, architecture, stack, and lessons with the
   verified result. Check `tabilet/evolution/` under its existing trigger;
   create no version for an implementation-only advance. Revisit affected
   candidates; a satisfied trigger still requires a new approved proposal.
5. Reconcile active downstream dependencies and acceptance. During an ordered
   `tabilet/GOAL.md` run, finish that protocol's downstream reconciliation
   before retirement. Consolidate knowledge and retire only after the review,
   verification, and downstream work pass.
6. Verify the resulting record and links, then commit substantive closure
   changes under the governing commit policy. Do not make an empty or
   redundant milestone-review commit. Report verification, review iterations,
   changed memory-bank files, any review commit, retirement record, durable
   lessons, and whether evolution changed.

## Long-Term Memory And Retirement

Retirement is agent work during the milestone closure above, not a background
process or a separate archive run. Completed, cancelled, and closed-historical
rows remain in their active status file until the whole milestone qualifies.
Old completed milestones are not retired merely because this procedure is
adopted; an explicit cleanup request and adequate closure evidence are needed.

The owner approved one explicit legacy preservation migration on 2026-09-24.
It moved 134 status files whose task rows were all terminal into the history
index, leaving blocked E15 and E18 active at that time. Both later completed
under the normal reviewed procedure; see their permanent history records. The
source status documents retain
their exact bytes in individual records; the complete pre-migration milestone
text is frozen at
[`milestone-before-legacy-retirement.md.txt`](../docs/history/milestone-before-legacy-retirement.md.txt),
SHA-256 `26884eeda9af4ded30e6d545a33dca84c401d2320c22355fe4fb0336d5dcac71`.
The source HEAD was `71a4f78afbcf2180fc478ffa89c53544c9160648`, with
uncommitted memory-bank changes included in the frozen files. Each record
binds both documents by SHA-256. `legacy-preserved` records reserve their IDs
and preserve history; they make no passing-review, delivered-acceptance, or
dependency-satisfaction claim. Normal future retirement still uses the
reviewed procedure below. The legacy path cannot be used for a new closure.
The separately tracked API runner validates the legacy envelope and frozen
snapshot at skills commit `dc674b876f0f93636f0e3154f731baeb199eb044`.

### Consolidate Before Retiring

Keep current facts in product, architecture, and stack documents. Maintain
[lessons.md](lessons.md) for concise, applicable, evidence-backed learning;
merge duplicates and retain useful lessons after their source milestone closes.
Before materially removing or superseding knowledge in those documents or
lessons, append its old wording to `tabilet/docs/history/knowledge.md` under a
unique dated heading. Include the source document and heading, reason,
supporting evidence, and replacement reference or reason none exists. Preserve
the old excerpt in a fenced Markdown block; append corrections rather than
rewriting old entries. Create and link the journal from the history index only
when needed. Routine wording edits require no journal entry. A context archive
is optional and never a retirement prerequisite.

### Retire A Closed Milestone

1. Require a passed review within the persisted ten-iteration limit, recorded
   verification, consolidated knowledge, and reconciled downstream work. No
   `[ ]`, `[~]`, or `[!]` row may remain. Every `[-]` row names its accepted
   successor. A cancelled or superseded milestone needs an authorized outcome;
   do not describe it as delivered acceptance.
2. Create `tabilet/docs/history/status-<LANE><NN>.md` with the envelope below.
   Preserve the complete final milestone specification and status document in
   separate literal Markdown fences, including all task text, notes, acceptance,
   and review evidence. Choose fences longer than any source fence. Relative
   paths inside the literal documents retain their source-document meaning.
3. Add one row to `tabilet/docs/history/index.md` with columns `Milestone |
   Outcome | Retired | Record | Summary`. Its ID, outcome, UTC date, and record
   link must match the envelope. Outcomes are `completed`, `cancelled`, or
   `superseded`; link to the record relative to that index.
4. Validate every envelope field and compare both retained documents with the
   complete active sources before removal. Use the full output of
   `git rev-parse --verify HEAD` for Git provenance; never use an abbreviated
   log hash. If validation fails, keep the active sources intact. Then remove
   the active status file, specification section, and status-index row; keep a
   single history-index link above. Repair maintained incoming links and
   downstream dependencies. Frozen archive snapshots remain unchanged.
5. Refresh an existing disposable goal suggestion to remaining active work, or
   remove it when empty. Do not create a new suggestion or launch execution.
   Verify the record and links before handoff. After interruption, reconcile
   active sources and destination before continuing; never overwrite a retired
   record or permit duplicate IDs.

The envelope records one field per line before the two literal sections:

`````markdown
# Retired milestone <ID> - <title>

**Milestone.** <ID>
**Outcome.** <completed, cancelled, or superseded>
**Retired.** <YYYY-MM-DD in UTC>
**Source status.** tabilet/memory-bank/status-<ID>.md
**Source specification.** tabilet/memory-bank/milestone.md#<original-anchor>
**Evidence.** <full Git HEAD or unversioned>
**Worktree.** <clean, includes uncommitted changes, or unversioned>
**Review.** passed
**Review iterations.** <1 through 10>
**Verification.** <commands, results, and supporting evidence>
**Consolidated into.** <current-document and lesson links, or no current-truth change>

## Milestone specification

````markdown
<Complete final milestone specification, not a summary.>
````

## Status record

````markdown
<Complete final status document, not a summary.>
````
`````

The evidence commit identifies the observed HEAD, not a claim that uncommitted
changes are in that commit. Without Git, both provenance fields are
`unversioned`. For cancelled or superseded outcomes, add `**Disposition.**`
with the authority, reason, and dependency disposition. A superseded outcome
also requires `**Successor.**` naming its accepted successor. Retired records
and index entries are frozen; append corrections to the knowledge journal or
open remediation work with a backlink rather than rewriting old records.

Retrieve current work from the active memory bank first, then search the
history index by ID or topic. A stale status path resolves through the retired
record's permanent ID and original-path metadata; a missing ID does not permit
recreation. A completed record can satisfy a dependency only when its recorded
acceptance matches the required outcome. Cancelled or superseded records need
their disposition and successor checked against current implementation.
Retirement follows the existing commit policy, including `GOAL.md`'s explicit
`COMMIT_POLICY: none` exception.

## Candidate Directions

Candidates have no lane, ID, status file, or execution-order entry until a
fresh scope and dependency review promotes them.

| Direction | Why Deferred | Promotion Trigger |
|---|---|---|
| S2b: simulation, supervised browser acquisition, and package-level pending steps | M87 delivers only S2a step commands. Tier-1 simulation, `browser acquire` checkpoints, snapshot checks, and pending-step approval rejection depend on separately owned UWS S1 and Browsertools contracts. | Kinet's stage-5/W04 scope is approved, required UWS mock/pending-step and Browsertools contracts exist, and an OpenUdon-owned command/approval integration outcome is explicitly approved. |
| S3: retire iCoT terminal/UI and shared interactive loops | Existing consumers, including W8M and Ramen, still rely on these surfaces. M87 is additive and does not authorize removal. | Replacement journey coverage is qualified, W8M migrates off OpenUdon UI launch paths, and removal is explicitly approved; Authoring retires its generic loops only after both OpenUdon and Ramen migrate under its compatibility policy. |
| Further package/source-family integration | A03/P01/A04/E01/E02 own the approved Browsertools authoring/evidence integration; other API/event source metadata remains owned by apitools and public semantics by UWS. | Another upstream contract is published and an OpenUdon-owned package/review outcome beyond this sequence is explicitly scoped. |
| Automated real-provider release evidence | Provider runs spend quota and can produce sensitive output; current policy remains local/manual. | Protected credentials, redaction, retention, spend bounds, and review-required CI policy are approved. |
| Trusted-runner capability expansion | OpenUdon hands approved packages to an external executor and must not absorb runtime semantics. | A public handoff/evidence gap is demonstrated without importing private runtime behavior or weakening approval gates. |
| UI-owned LLM drafting | The primary UI now owns deterministic acquisition, interview, review, and handoff, while extractor drafting and repair remain terminal/external-orchestration functions. | A separately reviewed engine mutation can invoke an optional extractor under exact-revision protection, persist every proposal as confirmation-required state, and preserve the no-silent-acceptance boundary. |

## Notes

- Historical full real-LLM smoke baseline in README: 2026-04-28, `gemini-2.5-flash`, structured
  output path, ten original examples passed with zero legacy extraction fallbacks. Current local
  real-LLM defaults use `copilot-api` with `gpt-5.4-mini`.
- M35-M37 retain historical local release-candidate and product-smoke evidence.
  No corresponding public GitHub tag or release exists; M69 owns the first
  public tag as v0.1.0. Real provider outputs remain ignored and local/manual.
- Advisory n8n reducibility fixtures remain part of the eval corpus. They should not introduce
  n8n-specific runtime behavior into OpenUdon or udon; use explicit intent, OpenAPI, and generic
  `fnct` or control-flow modeling.
- Normal deterministic gates remain `go test ./...`, `go vet ./...`, `make check`, and
  `git diff --check`. GitHub Actions runs provider-free public module gates and the repository
  boundary guard with `GOWORK=off`; sibling readiness, `check-doc-memory`, and real-provider release
  evidence remain local/manual.
- `openudon run` is the only OpenUdon-owned path that invokes a trusted executor runner, and it
  requires approval JSON plus a valid handoff package.
- OpenUdon no longer imports udon as a Go module; udon is an optional external trusted executor
  behind the run-config handoff.
- After a major review or milestone, check whether [tabilet/evolution/](../evolution/) needs a new
  prompt/result version.
