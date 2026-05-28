# OData Customers Query Eval

## Goal

Read a reviewed Customers entity set query using a package-local OData CSDL artifact

## Inputs

- `max_customers`: required number for the OData `$top` query option.

## Outputs

- `customers_result`: Customers query result from the trusted runtime.

## External Systems and OpenAPI

- Use `odata/odata-operations-service.xml` for OData source metadata.
- OpenAPI: none required.
- OpenUdon validates and packages the OData source-bound workflow, but OData execution belongs to the trusted runtime.

## Runtime Policy

- `openapi` and `http` are allowed for source-bound API operations.
- `fnct`, `cmd`, and `ssh` are not required.

## Function Contracts

- No `fnct` helper contracts are required.

## Data Flow

- Pass `max_customers` into the OData `$top` query option.

## Credentials and Secrets

- No credential values are stored in this provider-free fixture.
- Do not place OData tenant credentials, API keys, OAuth tokens, or other secret values in prompts, artifacts, or examples.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Real OData calls require human approval and trusted-runner execution.

## Fallback Behavior

- Stop if the OData source operation is unavailable or required query details are missing.
