# XRD-004 OpenAPI Eval Plan

XRD-004 is Ramen-owned until evals identify a reusable compiler or runtime gap. This plan expands
the curated eval corpus before any upstream `../udon` change is requested.

## Coverage Matrix

| Category | Fixture coverage | Notes |
| --- | --- | --- |
| Pagination | `paginated-list`, `customer-export-two-pages`, `cursor-pagination-report` | Covers fixed first page, two explicit pages, and cursor handoff from the first response to the second request. |
| Request bodies | `crm-note-write`, `order-fulfillment-chain` | Covers side-effectful POST bodies sourced from inputs and prior API responses. |
| Security schemes | `inventory-api-key-binding`, `cursor-pagination-report`, `order-fulfillment-chain` | Covers parameter-like API keys, OpenAPI bearer security, and per-service credential binding names. |
| Write operations | `crm-note-write`, `order-fulfillment-chain` | Covers sandbox write policy and trusted-runner approval evidence. |
| Response extraction | `weather-toronto`, `support-email`, `cursor-pagination-report`, `order-fulfillment-chain` | Covers array/object response paths, cursor extraction, and nested body fields feeding downstream requests. |
| Multi-service chains | `weather-toronto`, `order-fulfillment-chain` | Covers chained operations across one OpenAPI document and step-local operations across multiple OpenAPI documents. |

## Fixtures Added For XRD-004

- `cursor-pagination-report`: fetches two audit-event pages using a bearer-protected OpenAPI
  operation and binds `received_body.page.nextCursor` into the second request.
- `order-fulfillment-chain`: reads customer and inventory data from separate OpenAPI documents,
  then posts a fulfillment order request body to a third OpenAPI document.

## Acceptance

- Ramen has eval fixtures for pagination, request bodies, security schemes, write operations,
  response extraction, and multi-service chains.
- Fixtures stay under `examples/eval` and are documented in `docs/eval-gallery.md`.
- Ramen does not change prompt behavior, UWS semantics, udon runtime behavior, or automation for
  this pass.
- Any future eval failure that points to generic OpenAPI compilation or runtime behavior becomes a
  concrete upstream `../udon` follow-up.
