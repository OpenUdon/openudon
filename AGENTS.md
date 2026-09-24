# AGENTS.md

Requested features, candidate promotions, future direction changes, and new
engineering reviews follow the approved planning procedures in
[milestone.md](tabilet/memory-bank/milestone.md). Inspect and propose the complete
file actions before changing plans; implementation is a separate step.

## Purpose

OpenUdon is the public UWS workflow authoring, review, package, and executor-handoff tool. It can be
used directly or under optional external orchestration, and it hands approved packages to a
trusted executor boundary such as the private `udon` runtime.

OpenUdon owns project templates, optional workflow orchestration policy, example artifacts, validation
wrappers, review evidence, package digests, and trusted execution glue.

## Memory Bank First

The tracked canonical OpenUdon harness snapshot lives in
`../tofu/openudon`. In a normal `../openudon` checkout, `AGENTS.md`,
`tabilet/memory-bank/`, and `tabilet/evolution/` may be symlinks to this tracked snapshot so
agents can keep using the usual local paths while planning history is committed
in the `../tofu` repository.

Before making substantial changes, read in this order:

1. [tabilet/memory-bank/product.md](tabilet/memory-bank/product.md)
2. [tabilet/memory-bank/architecture.md](tabilet/memory-bank/architecture.md)
3. [tabilet/memory-bank/tech-stack.md](tabilet/memory-bank/tech-stack.md); consult
   relevant topics in [lessons.md](tabilet/memory-bank/lessons.md)
4. [tabilet/memory-bank/milestone.md](tabilet/memory-bank/milestone.md)
5. The matching active `tabilet/memory-bank/status-<LANE><NN>.md` for the work.

Use the memory bank as the active project source of truth. `tabilet/memory-bank/milestone.md`
owns the current-state dashboard, milestone sequencing, and status-file index.
Per-milestone task state lives in `tabilet/memory-bank/status-<LANE><NN>.md`. Do not recreate
duplicate root-level product, architecture, roadmap, or aggregate status
documents.
When retired history exists, consult its index and records only for an old ID,
dependency, or historical question. Current facts remain in the memory bank.
The 2026-09-24 `legacy-preserved` records are a one-time exact-source migration:
they reserve IDs and preserve evidence, but do not establish review, acceptance,
or dependency completion. E15 was left active by that migration while blocked,
then completed under the normal reviewed closure procedure. E18 remains active
because it still has blocked rows. Find E15 by ID in the history index; E18
remains in the memory bank. Future retirement follows the normal reviewed
closure gate in milestone.md.

This project exposes [tabilet/GOAL.md](tabilet/GOAL.md), one optional protocol for goal requests
that span multiple status files. Follow it only when a request names it.

A `tabilet/GOAL.md` run is a deliberate exception to the row-level commit rule below.
For that run, `COMMIT_POLICY: none` — the protocol default — means no commits,
while `COMMIT_POLICY: task` keeps the usual one-commit-per-row cadence.
Precedence is the request, then `tabilet/GOAL.md`, then this file; only commits are
delegated, and only during the run.

## Boundaries

- `../uws` is the public UWS specification and Go model. Put public workflow semantics there.
- `../udon` is the private UWS/OpenAPI compiler and runtime. Put generic execution/compiler capabilities there.
- OpenUdon source code must not import `../udon`, udon's private build-time siblings, or any private
  `genelet/*` executor module. OpenUdon invokes udon only as an external CLI or Docker executor through
  the trusted run-config handoff.
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
go test ./...
go run ./cmd/openudon check
go run ./cmd/openudon check-apitools-boundary
(cd tabilet && go run ../cmd/openudon check-doc-memory)
go run ./cmd/openudon validate ./examples/uws-validation
make check
```

## Architecture

Natural-language project brief -> operator task or external workspace -> Codex-generated OpenAPI/UWS artifacts -> validation/review -> approved artifact -> udon execution by trusted runner.

Agents may generate and validate artifacts. Production side effects must only happen through an approved trusted runtime path.

## Go Conventions

- Primary language is Go.
- Keep `cmd/openudon` thin.
- Put reusable logic under `internal/`.
- Keep scripts small wrappers around Go behavior when possible.
- Do not add product-specific behavior to `../uws` or core `../udon`.

## Safety

- Do not put secrets in prompts, examples, or committed workflow artifacts.
- Treat generated UWS/OpenAPI/HCL as untrusted until validated.
- Do not execute side-effectful workflows unless explicitly requested.
- Prefer sandbox/test endpoints for proof runs.

## Documentation Rules

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

## Development browser checks

Use `make fast` for routine iterations and one authorized affected `make smoke`
for UI/runtime changes. Default smoke is the typed registration UI-to-runtime
flow. Full `make qualify` is reserved for integration candidates and runtime
adoption; do not rerun it for every edit. Native qualification remains three
fresh repeats and never consumes the development cache. Reused development
results retain their original execution identity/time and cannot qualify a
runtime. See `docs/browser-system-eval.md` for cache and timing controls.
