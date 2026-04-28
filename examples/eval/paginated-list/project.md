# Paginated List Eval

## Goal

List the first page of customer records with a page size of 50.

## Inputs

- Page and limit are fixed by the goal.

## Outputs

- Customer list response.

## External Systems and OpenAPI

- Use `openapi/customers.yaml`.

## Runtime Policy

- `openapi` and `http` are allowed.
- `fnct` is allowed only for trusted adapters.
- `cmd` and `ssh` are not allowed.

## Data Flow

- Pass literal page `1` and limit `50` to the list operation.

## Function Contracts

- No function steps are expected.

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate and validate artifacts only.

## Fallback Behavior

- Stop if customers cannot be listed.
