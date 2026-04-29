# Inventory Reorder Decision

## Goal

Fetch inventory for a SKU and render a reorder decision from the returned quantity and threshold.

## Inputs

- `sku`: required string containing the SKU.

## Outputs

- `decision`: reorder decision payload.

## Data Flow

- Pass `inputs.sku` to `get_inventory`.
- Bind `get_inventory.received_body.sku`, `get_inventory.received_body.quantity`, and `get_inventory.received_body.reorderThreshold` into `decide_reorder`.

## External Systems and OpenAPI

- Inventory API: use `openapi/inventory.yaml`.
- The selected OpenAPI operation is `getInventory`.

## Runtime Policy

- `openapi` and `http` are allowed for inventory lookup.
- `fnct` is allowed for local reorder decision logic.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `decide_reorder`
  - Inputs: sku, quantity, and reorderThreshold.
  - Outputs: reorder decision.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required for this sandbox fixture.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Use sandbox inventory data for proof runs.

## Fallback Behavior

- Stop if inventory cannot be fetched.
