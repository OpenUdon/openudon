# Status A01 - API-First Browser-Profile Fallback Authoring

## Goal

Extend adaptive iCoT v2 and the package lifecycle to author UI-only workflows
from verified Browsertools artifacts without weakening API preference or the
trusted execution boundary.

## State

Completed.

## Dependencies

- UWS-B01 browser-profile interoperability baseline.
- Evidence A01 descriptor and lifecycle records.
- Browsertools M19 discovery and static registry clients.
- Udon M26 browser-profile runtime-plan and execution contract.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| Add CLI and v2 wire inputs | `[+]` | Added repeatable `--browser-profile` and `--browser-registry`, generic durable browser source records, safe digest/lifecycle/action/origin metadata, agent report fields, and existing v1 rejection. |
| Integrate local source discovery | `[+]` | Explicit profiles, examples, capability bundles, and source roots use Browsertools discovery; ambiguity/truncation block the run, mixed API/profile roots avoid cross-family false ambiguity, and Apitools remains unchanged. |
| Implement API-first fallback planning | `[+]` | Capability scoring selects a matching API operation first and makes a browser profile eligible only for an API gap or explicit reviewed selection; candidate workflows receive no technical breakdown. |
| Add browser frontier nodes | `[+]` | Source precedes action; action precedes mappings, opaque session posture, exact per-step mutation approval, outputs, fallback, and verification; generic full/normal/fast and agent contracts remain intact. |
| Add permission-gated registry lookup | `[+]` | Local and bounded HTTPS Browsertools registries use a browser-specific approval node; saved approval is reused during complete-session revalidation, while deny, timeout, unsafe-host, empty, invalid, and stale results remain visible deferable blockers independent of API lookup approval. |
| Extend proposal and atomic materialization | `[+]` | Proposals show profile origin/action/digest/lifecycle/session/approval evidence and exact writes; profile freshness is rechecked immediately before materialization, and profiles plus safe `.icot/browser-sources.json` metadata share existing collision, backup, rollback, draft, resume, and promotion transactions. |
| Extend intent, build, lint, and package safety | `[+]` | Browser action mappings lower to UWS 1.5, profile outputs drive response-path checks, package/review/handoff digests include profile evidence, and quality rejects secrets/raw shapes, expiry, tamper, invented actions, unsafe origins, and unconfirmed mutations. |
| Add evals, replay fixtures, and docs | `[+]` | Added the provider-free `browser-status-read` author/build fixture, local/HTTPS/API-preference/mutation tests, human review evidence, CLI/session/safety/operator docs, and evolution v12 with the static-catalog/no-membership boundary. |
| Run release and consumer gates | `[+]` | Workspace and standalone tests/vet, focused races, `make check`, doc-memory/boundary, variants, scorecard/report verification, strict docs, release SaaS, UWS/Evidence/Browsertools/Udon gates, shell syntax, and diff checks pass. OpenUdon now pins the committed Evidence, UWS, and Browsertools pseudo-versions without a local replacement. |

## Acceptance

- [x] OpenUdon can author a complete browser-bound UWS workflow from a verified local or static-registry profile.
- [x] An adequate API source wins unless the user explicitly selects the browser route.
- [x] No browser lookup, materialization, or execution bypasses network, lifecycle, proposal, approval, package, or side-effect gates.
- [x] OpenUdon owns no browser session, credential, driver, raw capture, or registry service behavior.
