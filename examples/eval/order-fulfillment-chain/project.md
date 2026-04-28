# Order Fulfillment Chain

## Goal

Create a sandbox fulfillment order after checking customer shipping details and inventory
availability across separate APIs.

## Inputs

- `customerId`: required string.
- `sku`: required string.
- `quantity`: required integer.

## Outputs

- `order_confirmation`: confirmation number from the created fulfillment order.

## External Systems and OpenAPI

- Customer API: use `openapi/customers.yaml`.
- Inventory API: use `openapi/inventory.yaml`.
- Fulfillment Order API: use `openapi/orders.yaml`.
- OpenAPI is required for all API operations.

## Data Flow

- Pass `inputs.customerId` to `get_customer.customerId`.
- Pass `inputs.sku` to `check_inventory.sku`.
- Pass `inputs.quantity` to `create_fulfillment_order.quantity`.
- Pass `get_customer.received_body.email` to `create_fulfillment_order.customerEmail`.
- Pass `get_customer.received_body.defaultShippingAddress.id` to `create_fulfillment_order.shippingAddressId`.
- Pass `check_inventory.received_body.sku` to `create_fulfillment_order.sku`.
- Pass `check_inventory.received_body.preferredWarehouseId` to `create_fulfillment_order.warehouseId`.

## Runtime Policy

- `openapi` and `http` are allowed for the Customer, Inventory, and Fulfillment Order APIs.
- `cmd`, `ssh`, and non-HTTP runtimes are not allowed.

## Credentials and Secrets

- Use credential binding `customers_bearer_token` for the Customer API.
- Use credential binding `inventory_api_key` for the Inventory API.
- Use credential binding `orders_bearer_token` for the Fulfillment Order API.
- Never include literal credential values in generated artifacts.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- The fulfillment order creation workflow is side-effectful and must only run through an approved trusted runner.
- Use sandbox Fulfillment Order API endpoints for proof runs.

## Fallback Behavior

- Stop if customer details cannot be fetched.
- Stop if inventory availability cannot be confirmed.
- Stop if the sandbox fulfillment order cannot be created.
