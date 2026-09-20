# M39 - iCoT Request Mapping Repair And Decision Evidence

## Goal

Improve iCoT at the most common intent-generation weak points: request mapping, bounded repair, and
confidence/evidence-driven blocking.

## Task State

| Item | State | Notes |
|---|---|---|
| Qualified request field prompt context | `[+]` | Request-mapping prompts expose known fields such as `path.ticketId` while preserving validated persisted field keys. |
| Mapping validation hardening | `[+]` | LLM-suggested unknown fields are rejected before entering intent state; qualified aliases normalize only when they map to declared fields. |
| Bounded repair command | `[+]` | `icot repair` supports JSON output, `--dry-run`, `--max-attempts`, and writes only through existing atomic artifact paths. |
| Repair scope enforcement | `[+]` | Repair is limited to request mappings, output sources, and dependency wiring; source, operation, credential, side-effect, and runtime/profile changes are rejected. |
| Decision evidence compatibility | `[+]` | Existing low/conflict decision evidence remains readiness-blocking; agent mode reports those blockers instead of prompting. |
| Documentation | `[+]` | Data-flow and iCoT docs describe qualified mapping, repair scope, and JSON inspection. |

## Acceptance Evidence

- Focused tests cover qualified request mapping aliases and repair dry-run JSON.
- Post-review hardening adds focused coverage for deterministic `depends_on` repair from step-output
  references and rejection of invented request fields in `icot repair`.
- Existing iCoT prompt-mode, replay, seed/build, and lint behavior remains compatible.
- Repair remains provider-free and does not call live providers or trusted executors.
- Full provider-free scorecard remained green after request-mapping and repair changes:
  56 total, 56 passed, 0 failed.

## Notes

- M39 keeps `--review-repair` as the interactive pre-final review repair path and adds `icot repair`
  as the deterministic command surface for existing examples.
- Further natural-language corpus expansion remains deferred to the corpus/provider roadmap, not this milestone.
