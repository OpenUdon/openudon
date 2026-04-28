# Customer Export Two Pages

## Goal

Fetch the first two pages of customer records and render one export payload.

## Outputs

- `export_payload`: merged customer export from both pages.

## External Systems and OpenAPI

- Customer API: use `openapi/customers.yaml`.
- OpenAPI is required for customer list retrieval.

## Data Flow

- Fetch page 1 with literal `page = 1` and literal `limit = 50`.
- Fetch page 2 with literal `page = 2` and literal `limit = 50`.
- Pass `list_customers_page_1.received_body.customers` to `merge_customer_pages.page_1`.
- Pass `list_customers_page_2.received_body.customers` to `merge_customer_pages.page_2`.

## Function Contracts

- `merge_customer_pages`
  - Inputs: customer arrays from page 1 and page 2.
  - Outputs: combined customer export payload.
  - Side effects: none.

## Runtime Policy

- `openapi` and `http` are allowed for the Customer API.
- `fnct` is allowed for local export rendering.
- `cmd` and `ssh` are not allowed.

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Do not write files or send the export anywhere.

## Fallback Behavior

- Stop if either page cannot be fetched.
- Stop if no approved merge function runtime exists.
