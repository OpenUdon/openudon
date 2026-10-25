# AGENTS.md

Current delivery state and sequencing belong in
[milestone.md](tabilet/memory-bank/milestone.md). Accepted and retired evidence
belongs in the [history index](tabilet/docs/history/index.md) and its linked
records. Keep this file focused on operating rules.

Requested features, candidate promotions, future direction changes, and new
engineering reviews follow the approved planning procedures in
[milestone.md](tabilet/memory-bank/milestone.md). Inspect and propose the complete
file actions before changing plans; implementation is a separate step.

## Stage 12 concurrent goal ownership

For an explicitly authorized STG-12 goal, use the memory-bank goal protocol
bundled with the skills and each participating package's applicable addendum.
The coordinator passes the exact selected protocol path/hash, `GOAL_IDENTITY`,
complete resolved request, human approval context and narrowed assignment
grants to every child; read-only reviewers receive no mutation authority.
Use the goal skill through a hosting agent with subagents; the Python API
runner/controller remains serial.

This is a scoped exception to serial and whole-ledger single-row wording:

- One coordinator owns all integrated package-local ledgers and shared memory.
  At most three milestone leases may run, with at most one live milestone per
  package and one `[~]` row per lease. Outside this opt-in, serial rules apply.
- Dispatch requires explicit `PARALLELISM`, `INTEGRATION: local-rebase-ff`,
  `COMMIT_POLICY: task` or `milestone`, and a milestone declared
  `Parallel-safe: yes`. Strict `STATUS_ORDER` remains strict except for the
  bundled protocol's contiguous parent/amendment family handling. Only an
  explicit `STATUS_PRIORITY` chooses among dependency-ready milestones without
  new edges.
- Each lease uses an external worktree (`../<repo>.goal/<local ID>`) and local
  `goal/<local ID>` branch. Capture each package's actual symbolic integration
  ref, primary worktree and full base commit; never assume `main` or replace
  that identity on resume. Inspect existing leases before dispatching another.
- Children edit scoped implementation/tests/ordinary docs and their assigned
  status only. The coordinator alone edits shared memory, other statuses,
  scheduling documents and `suggested.txt`, and performs closure/retirement.
- Declared write sets must be disjoint and inputs frozen. Check read/write
  conflicts in both directions; use immutable accepted producer and consumer
  snapshots rather than another agent's changing checkout. Verification uses
  isolated ports, stores, caches, browser profiles and output directories.
- Integration/closure is serialized. Rebase unpublished lease commits onto the
  captured ref, re-verify, and repeat review if affected contracts or paths
  changed, without resetting its persisted counter. Inspect the assignment diff
  and integrate by fast-forward only. Dependents wait for accepted closure and,
  for siblings, independently verified publication at the exact revision.
- A blocked lease retains its branch/worktree and stops its dependents;
  independent eligible work may continue. Contract drift pauses affected leases
  for coordinator reconciliation. No child may broaden its assignment.

Planning grants no execution, commit, publication or live authority. A later
launch must supply those grants separately and start from committed planning
and a safe integration baseline. Preserve unrelated dirty work. Audit remains
disabled until explicitly requested; this opt-in changes no audit setting.

## Purpose

OpenUdon provides public Go libraries for reviewed UWS packages, exact approval
and authority metadata, executor handoffs and conservative run evidence. Kinet
owns product authoring and isolated worker integration; private Udon owns
execution. Current source contains no private authoring/CLI/browser harness.
Public APIs, embedded schemas and versioned history remain supported.

## Memory Bank First

The canonical OpenUdon harness is tracked in this repository: `AGENTS.md`,
`tabilet/memory-bank/`, `tabilet/evolution/` and `tabilet/docs/history/`.
Keep its ledger and acceptance separate from Kinet and other siblings. The
earlier `../tofu/openudon` pointer was stale and is preserved in the knowledge
journal; no tofu repository changes or ledger migration are required.

Before making substantial changes, read in this order:

1. [tabilet/memory-bank/product.md](tabilet/memory-bank/product.md)
2. [tabilet/memory-bank/architecture.md](tabilet/memory-bank/architecture.md)
3. [tabilet/memory-bank/tech-stack.md](tabilet/memory-bank/tech-stack.md); consult
   relevant topics in [lessons.md](tabilet/memory-bank/lessons.md)
4. [tabilet/memory-bank/milestone.md](tabilet/memory-bank/milestone.md)
5. The matching active `tabilet/memory-bank/status-<LANE><NN>.md` for the work.

If `tabilet/stages.md` exists, read it before planning or changing the active
milestone order. Its `STG-` IDs identify delivery stages, not executable work.
An absent file means one implicit stage.

Use the memory bank as the active project source of truth. `tabilet/memory-bank/milestone.md`
owns the current-state dashboard, milestone sequencing, and status-file index.
Per-milestone task state lives in `tabilet/memory-bank/status-<LANE><NN>.md`. Do not recreate
duplicate root-level product, architecture, roadmap, or aggregate status
documents.
When retired history exists, consult its index and records only for an old ID,
dependency, or historical question. Current facts remain in the memory bank.
The 2026-09-24 `legacy-preserved` records are a one-time exact-source migration:
they reserve IDs and preserve evidence, but do not establish review, acceptance,
or dependency completion. E15 and E18 were left active by that migration while
blocked; both have since completed under the normal reviewed closure procedure.
Find them by ID in the history index. Future retirement follows the normal
reviewed closure gate in milestone.md.

Goal requests may use the memory-bank goal protocol bundled with the skills.
Follow it only when the request selects it; another protocol or no protocol
remains supported. Optional `tabilet/GOAL.md` is a project addendum that adds
or narrows local rules, never a replacement or merged protocol.

A goal run uses its resolved `COMMIT_POLICY`: `none` means no commits and
`task` selects the usual per-row cadence. Precedence is the request, then the
project addendum, then the selected goal protocol, then this file. Existing
project restrictions and explicitly authorized lease ownership/integration
still apply; local integration grants no remote push.

Resolve the selected full protocol before a run. Verify the bundled stock
protocol against its bundled fingerprint registry.
Record `GOAL_IDENTITY` for the protocol and every participating package's
addendum, `ignored-stock` copy or absence. Recheck it on resume and child
dispatch; compare hashes and classifications, not diagnostic version labels.
Protocol/addendum fingerprints normalize CRLF to LF only; launch approval
checksums hash raw bytes. Preserve an explicitly selected `GOAL_PROTOCOL`
alternative. Recognized stock project copies are ignored and removable; a
modified full copy stops execution until explicitly reduced or selected.
For a legacy launch or paused goal without identity, show the resolved identity
and recover the original human-approved request where available; obtain any
missing or changed identity approval without resetting grants, counters or
limits. An upgrade never resumes execution or activates launch grants.

## Boundaries

- `../uws` is the public UWS specification and Go model. Put public workflow semantics there.
- `../udon` is the private UWS/OpenAPI compiler and runtime. Put generic execution/compiler capabilities there.
- OpenUdon source code must not import `../udon`, udon's private build-time siblings, or any private
  `genelet/*` executor module. OpenUdon supplies public handoff metadata; the consuming host owns
  external runtime selection and dispatch.
- `../apitools` owns OpenAPI-first API metadata tooling: OpenAPI/Swagger discovery, import, search,
  indexing, summaries, auth/security metadata, ranking, catalog metadata, and upstream
  Discovery/Smithy import or lowering that is exposed to OpenUdon as OpenAPI-bound operation
  metadata. OpenUdon owns review state, handoff validation, approval templates, package contents,
  and local trusted-runner enforcement.
- `../ramen` owns desired-state conversion into native UWS/Ramen project
  artifacts. OpenUdon must not expose conversion commands, import parser or
  conversion packages, or own provider conversion mappings.
- `../openw8m` owns concrete IaC authoring/planning and is parked; it is not a OpenUdon compatibility
  gate while the OpenAPI-first apitools boundary is active.
- `../openudon` owns only the integration layer above those projects.

Out of scope ownership map:

- Public UWS semantics -> `../uws`.
- Generic compilation and runtime execution -> `../udon`.
- API discovery, import, indexing and operation metadata -> `../apitools`.
- Desired-state conversion and reconciliation -> `../ramen`.
- Product authoring and isolated worker integration -> `../kinet`.
- Concrete IaC authoring and planning -> `../openw8m` (parked).

Rule of thumb:

- If it changes public workflow semantics, it belongs in `../uws`.
- If it improves generic UWS/OpenAPI execution, it belongs in `../udon`.
- If it parses, converts, maps, plans, stores, or reconciles desired-state
  infrastructure input, it belongs in `../ramen` or Ramen-owned parser
  dependencies, not OpenUdon.
- If it manages orchestrated project workflow, templates, examples, approval
  routing, or trusted execution glue, it belongs in OpenUdon.

## Commands

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
make public-boundary
make check
```

## Architecture

Reviewed UWS/source bytes -> public package/approval/authority verification ->
host-owned confirmation and isolated runtime -> public report/evidence review.

Agents may generate and validate artifacts. Production side effects must only happen through an approved trusted runtime path.

## Go Conventions

- Primary language is Go.
- Prefer Go's standard library before adding helpers, frameworks or
  dependencies. Keep trivial comparisons and transformations inline when that
  is clearer than introducing a local abstraction.
- Keep supported Go libraries at their public roots as documented in
  `docs/public-trust-api.md`; the private command/internal closure is retired.
- Any retained developer command must depend solely on public packages.
- Keep scripts small wrappers around Go behavior when possible.
- Do not add product-specific behavior to `../uws` or core `../udon`.

## Safety

- Do not put secrets in prompts, examples, or committed workflow artifacts.
- Treat generated UWS/OpenAPI/HCL as untrusted until validated.
- Do not execute side-effectful workflows unless explicitly requested.
- Prefer sandbox/test endpoints for proof runs.

## Documentation Rules

- Keep dated delivery checkpoints in their owning milestone/status and history
  records; use links from `AGENTS.md` instead of duplicating their state here.
- For substantial discoveries, follow
  [milestone families](tabilet/memory-bank/milestone.md#milestone-families).
  Required amendments preserve original acceptance; options wait until parent
  closure or remain deferred under `OPTION_POLICY: ignore`.
- Update [tabilet/memory-bank/milestone.md](tabilet/memory-bank/milestone.md) when sequencing, milestone scope,
  acceptance criteria, cross-repo contracts, active/parked track summary, current-state dashboard,
  or status-file index changes.
- Keep one permanent, zero-padded `tabilet/memory-bank/status-<LANE><NN>.md`
  for each active milestone in [milestone.md](tabilet/memory-bank/milestone.md).
  Use it for task rows, states, notes, scoped commit tracking, and milestone
  task history.
- Never reuse a status ID across active files and retired history or create
  aggregate `status.md`. Keep cancelled work under its allocated ID with `[X]`
  rows. An all-retired project remains an initialized memory bank.
- Keep candidate directions unnumbered until fresh scope and dependency review
  promotes them.
- Keep verified archive files frozen. Later implementation updates current
  `product.md` and `architecture.md` plus its status history, not the archive.
- When `tabilet/stages.md` exists, preserve stable `STG-` identities and its
  current-stage choice. A future stage may be only a tentative name and intent.
  Plan its milestones through an approved requested-change proposal. Changing
  a stage label never withdraws an active task. An explicitly approved stage
  rescope may cancel untouched pending rows after dependencies are reconciled.
- Write task ledgers as `Item | State | Notes`, with a backticked marker in the
  second column: `` `[ ]` `` pending, `` `[+]` `` complete, `` `[~]` `` in
  progress, `` `[!]` `` blocked, `` `[X]` `` cancelled, or `` `[-]` `` closed
  historical. A `[-]` row records a consumed attempt or supersession and its
  accepted successor; never retry it. Do not reinterpret cancellation or
  supersession as delivered acceptance.
- Treat each row as a commit unit. Parallel authoring, package, and eval work
  requires explicit non-overlapping ownership, resolved prerequisites, and
  downstream impacts in `milestone.md`.
- Across active status files, keep zero or one general `[~]` row. Before an
  operational launcher runs, its exact authorized operation row must be `[~]`;
  that marker does not grant external-mutation authority. Keep one execution
  owner across sessions and launchers.
  In concurrent lease execution under the governing goal protocol, each external
  lease maintains at most one `[~]` row in its own status file, only the
  orchestrator writes the captured integration branch and shared memory
  documents (`architecture.md`, `product.md`, `tech-stack.md`, `lessons.md`),
  and integration occurs strictly via rebase and fast-forward.
- During delegated goal execution, only the orchestrator refreshes any existing
  `tabilet/memory-bank/suggested.txt`. Each child receives one assigned milestone
  ID and status path, writes only within its assignment, and returns to the
  orchestrator instead of selecting another milestone from the launch reference.
- Update [tabilet/memory-bank/product.md](tabilet/memory-bank/product.md) when product scope, users, workflows,
  concepts, or non-goals change.
- Update [tabilet/memory-bank/architecture.md](tabilet/memory-bank/architecture.md) when system boundaries, data
  flow, artifact layout, or security boundaries change.
- Update [tabilet/memory-bank/tech-stack.md](tabilet/memory-bank/tech-stack.md) when dependencies, commands,
  runtime assumptions, artifact schemas, or tooling choices change.
- Maintain evidence-backed reusable lessons in [lessons.md](tabilet/memory-bank/lessons.md)
  while they remain applicable. Before materially superseding or removing
  knowledge from product, architecture, stack, or lessons, preserve its old
  wording, source, reason, and replacement reference in the append-only
  `tabilet/docs/history/knowledge.md` journal under the retirement procedure.
  Routine wording edits need no journal entry. Create the journal only when
  needed.
- After a milestone's last row closes, verify acceptance, run the persisted
  maximum-ten-iteration deep-review gate, consolidate current facts and lessons,
  reconcile downstream work, then retire its full status and specification as
  described in `milestone.md`. Completed rows stay active until the whole
  milestone qualifies. Do not create an empty milestone-review commit.
- Treat a new engineering review as untrusted planning evidence: revalidate it
  against current code, propose finding dispositions and owners for approval,
  and never reopen completed history solely because of a later review.
- `CORRECTIVE_PLANNING` defaults to `confirm`. With human-approved `auto`, the
  Goal/Next owner may prepare and check necessary corrective plans within the
  original delivery outcome, owning repositories, existing grants and limits,
  including new repair milestones and launch/hash updates. Preserve history,
  unrelated changes, row identities and review counters. Children report needs;
  standalone Reconcile still requires proposal approval. Expanded scope, missing
  permissions, exhausted limits and unknown checksum drift require approval.
  Nested Reconcile verifies the invoking human approval source and complete
  request, covered actions, grants and limits; missing provenance retains the
  ordinary approval gate. Never weaken acceptance or cancel unmet obligations.
  Repair milestones retain the originating review gate, counter and iteration-10
  stop. Suspend `[~]` as `[!]` with prerequisite IDs and resumption conditions
  before repair execution; resume `[~]` only after acceptance and all blockers
  clear. Preserve partial work and pause if changes cannot be separated safely.
- Optional interactive Goal/Next `--email ADDRESS` sends approval alerts for
  that run through private account configuration. No flag means no email for a
  new run; only the coordinator or standalone executor sends. Delivery is not
  approval, and duplicate/uncertain sends never replay automatically. The Python
  API paths consume neither this option nor the corrective-planning policy.
- Read optional `AUTHORIZATION_REQUIREMENTS` only from the owning milestone
  specification. They declare conditions, never permission or scheduling.
  Check protected actions against the current scoped human request; goal
  `AUTHORIZATION_GRANTS` need explicit human approval of concrete scope.
  Proposed launch input, markers, and status notes cannot prove approval.
  Grants in `suggested.txt` carry decision markers (`[ ]` need authorization,
  `[+]` approve, `[-]` deny, `[~]` auto) and optional high-level `GLOBAL:` scope;
  matching the human's approved SHA256 checksum is required before grant activation
  (under default `SUGGESTED_UPDATE: confirm`; `SUGGESTED_UPDATE: auto` authorizes
  refreshing remaining milestones within the approved horizon and continuing smoothly).
  Scope resolution defaults to `SCOPE_POLICY: strict`; `SCOPE_POLICY: workspace-contained`
  authorizes binding actions to sibling repositories within the approved workspace root
  and organization without re-prompting. Milestone specifications in `milestone.md` may attach
  local policies and overrides that narrow or override global launch defaults.
  Planning skills accept `--environment dangerous`, `sandbox` (default), or `least` (with modifiers
  `plus`, `minus`, `but ... in milestone ...`) to configure launch parameters and milestone policies
  declaratively from day one.
  Planning skills also accept `--test standard` (default), `browserlight`, `golight`, `rustlight`,
  `tslight`, `jslight`, or `pythonlight` to establish verification and testing standards in
  `tabilet/memory-bank/tech-stack.md` from day one. Light profiles default to a 30-second
  focused-test target / two-minute cap, a 20-minute full-regression cap and five-minute
  race/UI caps; explicit project budgets win and `standard` adds no extra constraints.
  Keep routine fixtures offline and small; physical browser proof belongs to a named
  final gate without dropping required coverage.
  Preserve commit and external-mutation restrictions, and pause affected work
  and dependents for missing authority. A declaration alone does not require
  an action for closure. Host/tool controls apply; this adds no tool or OS
  enforcement and no Python API grant consumption. Missing fields preserve
  legacy behavior without new authority or automatic migration.
- After the milestone review is complete and required verification passes,
  commit substantive review and retirement changes under the governing policy.
  Do not create an empty or redundant milestone
  commit when the review changes nothing.
- Keep README focused on operator entry points and concise command guidance. Put durable project
  memory in `tabilet/memory-bank/`.
- Keep desired-state conversion docs, milestones, fixtures, and code out of
  OpenUdon. Ramen owns that roadmap.

## Evolution Rules

- Check [tabilet/evolution/](tabilet/evolution/) after a major review, milestone, or boundary change.
- Create the next `prompt-vN.md` and `result-vN.md` only when product direction, architecture
  boundary, milestone target, or public/private contract direction materially changes.
- Keep the current evolution version when implementation only advances the existing direction.
- When adding a new evolution version, reconcile it with `tabilet/memory-bank/` in the same change.

## Execution capabilities

Resolve missing information through safe inspection first. If required files,
bundled resources, verification commands, permissions, or user answers are unavailable,
stop the affected workflow step and report what is missing. Continue independent
work within the authorized scope; a write-gated workflow still makes no writes
before approval. Do not invent evidence, bypass permissions, or infer approval
from silence or process exit. Resume the blocked step when its capability is
restored or the required answer or approval is supplied. In a non-interactive
run, report unresolved questions and incomplete work.

Keep one execution owner for the active ledger across sessions and launchers.
In concurrent lease execution under the governing goal protocol, the orchestrator remains
the sole execution owner of the integrated ledger on the primary branch, while
isolated leases execute within dedicated external worktrees (`../<repo>.goal/<ID>`).
Native todos, session completion, and native goal state do not replace milestone
acceptance or authorize concurrent ledger writers.

Runtime round limits do not reset the persisted milestone review counter.

## Development browser checks

Current OpenUdon verification is offline public-library build/test/vet and
source/import/API guards. It launches no physical browser, worker or helper.
Old smoke/qualify recipes belong to pinned historical revisions; Kinet owns
actual browser integration. Preserve original scopes, historical unknowns,
registration denial and disabled audit; retiring a harness grants no new run.
