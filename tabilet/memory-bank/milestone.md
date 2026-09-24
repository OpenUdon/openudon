# Milestone

## Current State

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
remains the default and v1/v2 reports retain their readers. E21.4 is preparing
the complete current-stack evidence from a clean committed source revision.

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

The history index holds 134 legacy-preserved status IDs and two normally reviewed completions. Legacy-preserved records retain exact source bytes and the frozen milestone text, but do not establish acceptance by themselves. Search the history by ID when needed.

## Active Milestone Specifications

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

- Active: E21 owns the current-stack report-preservation and Udon build-lock
  repair; W8M W20.6/W21 consume its published exact lock. No deployment,
  public canary, runtime adoption, or target operation is authorized.
- Parked: real-provider evidence, live W8M operation, and public canaries need
  separately approved scope and authority.
- Completed history: use the [history index](../docs/history/index.md) for
  terminal ID records and the frozen earlier milestone text.

## Status Files

Active status files are indexed below. The memory bank remains initialized;
search the history index before allocating a future ID.

| ID | Milestone | Status file | State |
| --- | --- | --- | --- |
| E21 | Repair current-stack Udon build and preserve M86 report meaning | `tabilet/memory-bank/status-E21.md` | Active |

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
