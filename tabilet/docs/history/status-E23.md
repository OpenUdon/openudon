# Retired milestone E23 - Current-stack native input identity

**Milestone.** E23
**Outcome.** completed
**Retired.** 2026-09-29
**Source status.** tabilet/memory-bank/status-E23.md
**Source specification.** tabilet/memory-bank/milestone.md#e23-current-stack-native-input-identity
**Evidence.** 75b6bcc19b3153c482b471134c872e05f378a26f
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Focused current-input, dependency/host and CLI tests passed; package vet, check-doc-memory and git diff --check passed. Pure browser-free input helper; no browser qualification or adoption claimed.
**Consolidated into.** [product](../../memory-bank/product.md), [architecture](../../memory-bank/architecture.md), [tech stack](../../memory-bank/tech-stack.md), and [browser-system guidance](../../../docs/browser-system-eval.md). W8M W27.2a consumes the completed helper.
**Specification SHA256.** 216e0cca7d9d643f156fdc9819f31308b80794f8cf695f859891b3da210d8f22
**Status SHA256.** 1d819fb30aaae3287b8106c441aa2997d403b952c8f87f57fa1d2b69763d6ec7

## Milestone specification

````markdown
## E23 — Current-stack native input identity

Approved prerequisite of W8M W27's candidate native-reuse implementation. Add an
explicit current-stack input v2 endpoint selecting the supplied external module
bundle; preserve the legacy v1 input and all qualification/report contracts.
Bind exact nineteen-source closure, paths, bytes/modes, tools/dependencies,
browser/Playwright, environment/display and host/boot/namespace identity without
exporting raw inputs or launching browsers. Validate readiness before hashing.
E23.1 owns implementation/tests and its local task commit; E23.2 owns review,
current-fact consolidation and W8M handoff. W8M owns caches and runtime adoption.
Acceptance/verification and the ten-iteration review are in status-E23.md.
````

## Status record

````markdown
# Status E23 — Current-stack native input identity

**State:** Complete. Approved through W8M W27's native-reuse proposal and owner
implementation approval on 2026-09-29. This helper emits input identity only;
it never executes browsers, creates a reusable cache or adopts a runtime.

| Item | State | Notes |
| --- | --- | --- |
| E23.1 Add current-stack and external-module input inventory | `[+]` | Preserve the legacy v1 endpoint. Add explicit current-stack input v2 with nineteen source inventories, exact roots, external modules/permissions, dependency/tool/browser/environment/host identities. Validate canonical paths, locked modules, executable npm entries and cancellation. Focused browser-free tests and vet; task-level local commit after acceptance. W8M W27.2a consumes its exact commit. |
| E23.2 Review and consolidate helper acceptance | `[+]` | Bounded whole-milestone review (ten-iteration maximum), docs/current-fact consolidation and handoff to W8M. No full qualification required for the pure inventory helper. No runtime adoption, seed execution, deployment or push. |

## Acceptance and dependencies

Existing QualificationInput and its v1 meaning remain intact. New current input
selection binds the real supplied dependency bundle without staging/building it
or running browsers. Changed native input bytes/modes/roots/host conditions
change identity or reject admission; unsupported selections fail closed. No raw
environment or display-authority values leave the API. No W8M-specific code or
private executor imports. W8M retains cache eligibility and fresh acceptance.

## Verification and review

Start with affected inventory, canonical-path, module readiness and CLI tests,
then package vet and memory-doc checks. Estimate: seconds to a few minutes for
focused browser-free checks; any actual full inventory is measured separately.
Whole-milestone review: passed, iteration 1/10; no open P1/P2. Earlier E/P history remains unchanged.

## Provenance and ownership

Baseline is the clean P08 HEAD; this isolated local branch preserves the original
P08 checkout and all frozen W8M evidence clones. AGENTS.md and the memory bank
are tracked here; the described ToFu/openudon default does not exist on this
host. The owner approved W8M's complete proposal including this source surface
and upstream planning intake. Required file actions: inputs.go, its inventory
helpers/tests, cmd/openudon/main.go and CLI tests, current memory-bank facts,
milestone/status and browser-system-eval guidance. Canonical verification stays
OpenUdon-owned; W8M's proposal governs no external operation. Local task commits
only, no pushes or source downloads.

E23.1 verification: focused inventory/host/dependency/current-module and CLI tests passed (0.658 s and 1.262 s). Package vet and check-doc-memory passed; git diff --check is clean. No evolution snapshot: this additive helper advances the existing input-inventory boundary. Source review checks legacy default dispatch, canonical bundle boundaries, byte/mode hashing, no browser launch and no source mutation.

E23.2 review iteration 1/10: reviewed full helper diff, legacy CLI dispatch, strict output versions, cancellation, external-bundle byte/mode and root binding, executable entry checks, environment privacy and source non-mutation. No P1/P2 finding. Focused tests, package vet, check-doc-memory and diff checks passed. W8M must pin the final E23 closure commit and still qualify its own candidate; helper acceptance grants no runtime adoption.
````
