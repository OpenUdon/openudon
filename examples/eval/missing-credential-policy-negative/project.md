# Missing Credential Policy Negative

## Goal

Render a local credential-policy gap report when a workflow request needs authentication but no approved credential binding is available.

## Inputs

- `apiName`: required string containing the API name.
- `operation`: required string containing the operation name.

## Outputs

- `credential_gap`: credential policy gap report.

## Data Flow

- Pass `inputs.apiName` and `inputs.operation` to `render_credential_gap`.

## External Systems and OpenAPI

OpenAPI: none required.

## Runtime Policy

- `fnct` is allowed for local credential policy reporting.
- `openapi`, `http`, `cmd`, and `ssh` are not allowed.

## Function Contracts

- `render_credential_gap`
  - Inputs: apiName and operation.
  - Outputs: credential policy gap report.
  - Side effects: none.

## Credentials and Secrets

- No credentials are declared for this negative fixture.
- Do not invent credential binding names or include secret values.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Do not execute authenticated API calls when credential policy is missing.

## Fallback Behavior

- Stop after rendering the credential gap report.
