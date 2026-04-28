# Inventory API Key Binding

## Goal

Fetch inventory availability for a requested SKU.

## Inputs

- `sku`: required string.

## Outputs

- `availability`: inventory availability response.

## External Systems and OpenAPI

- Inventory API: use `openapi/inventory.yaml`.
- OpenAPI is required for inventory lookup.

## Data Flow

- Pass `inputs.sku` to `get_inventory.sku`.
- Bind the required API key parameter from `inventory_api_key`.

## Runtime Policy

- `openapi` and `http` are allowed for the Inventory API.
- `cmd` and `ssh` are not allowed.

## Credentials and Secrets

- Use credential binding `inventory_api_key`.
- Never include the Inventory API key value in generated artifacts.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Use sandbox Inventory API endpoints for proof runs.

## Fallback Behavior

- Stop if `sku` or `inventory_api_key` is unavailable.
