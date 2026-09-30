# Status E23 — Current-stack native input identity

**State:** Active. Approved through W8M W27's native-reuse proposal and owner
implementation approval on 2026-09-29. This helper emits input identity only;
it never executes browsers, creates a reusable cache or adopts a runtime.

| Item | State | Notes |
| --- | --- | --- |
| E23.1 Add current-stack and external-module input inventory | `[+]` | Preserve the legacy v1 endpoint. Add explicit current-stack input v2 with nineteen source inventories, exact roots, external modules/permissions, dependency/tool/browser/environment/host identities. Validate canonical paths, locked modules, executable npm entries and cancellation. Focused browser-free tests and vet; task-level local commit after acceptance. W8M W27.2a consumes its exact commit. |
| E23.2 Review and consolidate helper acceptance | `[ ]` | Bounded whole-milestone review (ten-iteration maximum), docs/current-fact consolidation and handoff to W8M. No full qualification required for the pure inventory helper. No runtime adoption, seed execution, deployment or push. |

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
Whole-milestone review: not started, 0/10. Earlier E/P history remains unchanged.

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
