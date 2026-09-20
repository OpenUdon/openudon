# Status A05 - Live Browser Observation Hardening

## Goal

Verify Browsertools' canonical live labels and negotiated candidate authority
before any observation reaches a human display or planner.

## State

Complete.

## Dependencies

- OpenUdon A04/E03 live orchestration and exact seam qualification.
- Browsertools E06 commit `55f40e56e03e2ce878521a50bc38a3674e18defc`,
  published as `v0.0.0-20260817000231-55f40e56e03e`.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| Implement, document, pin, and verify live observation hardening | `[+]` | OpenUdon commit `7f4b255c64518c6301dfbae41a8d56a0d5014fb2` pins Browsertools E06, replaces duplicated label screening with canonicality validation, orders candidate checks, emits only safe ID/reason rejections, applies the negotiated 128 ceiling to observation length and match counts, and restricts planner navigation to canonical same-origin GET targets. Property/fake-session/boundary tests prove canonical outputs pass, every noncanonical reason fails safely, suspicious marker observations continue, rejected labels reach neither output nor planner, 128 passes, and 129 fails. Workspace/standalone tests and vet, `make check`, doc-memory, strict MkDocs, clean integration, and diff/link checks passed. |

## Evolution Review

No new evolution version: A05 hardens the existing live-browser boundary and
does not change product direction, public wire versions, CLI shape, or runtime
ownership.
