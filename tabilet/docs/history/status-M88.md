# Retired milestone M88 - Stage 1 step-authoring remediation

**Milestone.** M88
**Outcome.** completed
**Retired.** 2026-09-27
**Source status.** tabilet/memory-bank/status-M88.md
**Source specification.** tabilet/memory-bank/milestone.md#m88--stage-1-step-authoring-remediation
**Evidence.** 451ed7b77f85edd06b0c5805b8df871970eff3a4
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 2
**Verification.** Focused bind/check and generated-UWS regressions; `GOWORK=off go test ./... -count=1`; `GOWORK=off go test -race ./internal/stepauthoring ./cmd/openudon -count=1`; `GOWORK=off go vet ./...`; `make check`; `GOWORK=off go run ../cmd/openudon check-doc-memory` from `tabilet`; and `git diff --check` passed. Iteration 1's malformed-prefix fix passed affected tests, race, vet, and diff checks again.
**Consolidated into.** [architecture](../../memory-bank/architecture.md), [technical stack](../../memory-bank/tech-stack.md), and the [step-authoring v1 guide](../../../docs/step-authoring-contract-v1.md). No new product direction, reusable lesson, or evolution bump was needed.

## Milestone specification

````markdown
### M88 — Stage 1 step-authoring remediation

**Goal.** Correct the three confirmed OpenUdon P2 findings in the 2026-09-27 UWS/APItools/OpenUdon stage 1 review and consume APItools' corresponding M78 metadata fixes. M87 stays retired.

**Acceptance.** `step bind` and `step check` compare mapped request keys with source parameter locations, including body/query/path/header/cookie and colliding names. They compare actual workflow input type and requiredness with the selected contract/source requirement, returning fail or indeterminate when evidence is incomplete. Inline `credentials.<symbol>` values obey the same grammar and reserved-sentinel rejection as explicit credential bindings. APItools' repaired effect and nullable output metadata flow through the commands without a false compatible result. Rejected binds leave intent bytes unchanged. Existing v1 request/result shape, CLI exits, source digests, and UWS pin policy remain compatible.

**Order and downstream.** Published APItools M78 is a prerequisite for [M88.1–M88.4](status-M88.md). Rows run in order: dependency pin and consumer regression, location, mapped type, credential validation. Kinet W03 consumes the revised commands and fixtures; it owns its separate real-tool check. M88 does not change Kinet, UWS semantics, browser locks, live providers, or approval policy.

**Verification.** Run focused bind/check cases, schema and fixture conformance, `GOWORK=off go test ./...`, affected race tests, `GOWORK=off go vet ./...`, `make check`, `check-doc-memory`, `git diff --check`, and the persisted whole-milestone review gate before retirement and push.
````

## Status record

````markdown
# Status M88 — Stage 1 step-authoring remediation

**State:** Complete. Whole-milestone review passed in iteration 2.

**Provenance:** 2026-09-27 UWS/APItools/OpenUdon stage 1 review R2, R3, R6; each source P2 and local P2, confirmed against clean OpenUdon `c377c8d10e083fabe175b200b1c472f8a1b4dbc8`. Review baseline is the same commit; no uncommitted repository changes were part of the evidence. Published `docs/examples/step-authoring/v1` fixture variants reproduce all three. Historical owner M87 is retired.

| Item | State | Notes |
| --- | --- | --- |
| M88.1 Adopt APItools remediation | `[+]` | Pinned published APItools M78 `26bb05247d6c48f8ee60b9ae178f6ef6d48bbe3d`; compound read/mutation bind refusal and nullable output candidate regressions pass with `GOWORK=off go test ./internal/stepauthoring -count=1`. The v1 wire remains unchanged; current pin truth is updated in `tech-stack.md`. |
| M88.2 Enforce request locations | `[+]` | Shared bind/check validation now compares request mappings with source parameter and body locations, including path/query/header/cookie, and rejects colliding unqualified names. Wrong-location bind leaves intent absent; check fails on a stale wrong-location intent, and a build regression proves the wrong key would enter UWS `body`. Affected `stepauthoring`, CLI, and synthesis suites and `git diff --check` pass. Architecture current truth updated. |
| M88.3 Validate mapped workflow values | `[+]` | Direct `inputs.<name>` mappings now compare actual workflow input type and requiredness against the contract and source field; nested or unproven expressions report indeterminate, and bind refuses both failed and indeterminate evidence without mutation. Scaffold wrong-type/optional and read-only check regressions pass with `GOWORK=off go test ./internal/stepauthoring ./cmd/openudon -count=1` and `git diff --check`. Public guide and architecture current truth updated. |
| M88.4 Validate inline credential symbols | `[+]` | Shared symbol validation rejects reserved, empty, and malformed inline credential references before bind writes; check fails them in existing intent files, while a valid inline reference remains bindable. Iteration 1 also closed malformed slash/colon prefixes. Focused/full standalone suites, affected race tests, standalone vet, `make check`, `check-doc-memory`, and `git diff --check` passed. Public guide and architecture current truth updated. |

## Acceptance and review

All rows and M88 verification in `milestone.md` must pass before a whole-milestone review. Persist the review iteration here before each pass (maximum ten).

**Review iteration 1:** Started after all rows and required verification passed. Reviewed the full diff from `c377c8d10e083fabe175b200b1c472f8a1b4dbc8`, including the APItools pin, source location and type mapping, authentication alternatives, wrong-location generated UWS, public v1 fixture/schema compatibility, no-write refusal, documentation, and downstream Kinet handoff. Found one P2 gap in M88.4: `credentials/clear` and `credentials:clear` looked like malformed credential references but read-only check could treat them as ordinary symbolic auth values. Fix this before a second full pass.

**Review iteration 2:** Started after rejecting slash/colon credential prefixes and passing affected `stepauthoring`/CLI tests, race, vet, and diff checks. Reviewed the complete M88 diff again, including this fix; no P1/P2 issue remains.

**Review outcome:** Iteration 2 reviewed the complete implementation, fixtures, public guide, pin, source and auth boundaries, refusal paths, and downstream handoff. No P1/P2 finding remains. Current truth is consolidated in `architecture.md`, `tech-stack.md`, and `docs/step-authoring-contract-v1.md`. This corrects the approved v1 behavior without a new version or architecture direction, so no evolution bump or reusable lesson is needed. Kinet W03 remains a separate consumer and must run its own real-tool check against the published OpenUdon revision.
````
