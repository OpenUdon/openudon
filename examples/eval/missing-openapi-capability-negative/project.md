# Missing OpenAPI Capability Negative

## Goal

Record that a requested provider action has no available OpenAPI evidence and render a local gap report instead of inventing an API call.

## Inputs

- `provider`: required string containing the provider name.
- `action`: required string containing the missing action name.

## Outputs

- `gap_report`: local capability gap report.

## Data Flow

- Pass `inputs.provider` and `inputs.action` to `render_capability_gap`.

## External Systems and OpenAPI

OpenAPI: none required.

## Runtime Policy

- `fnct` is allowed for local gap reporting.
- `openapi`, `http`, `cmd`, and `ssh` are not allowed.

## Function Contracts

- `render_capability_gap`
  - Inputs: provider and action.
  - Outputs: capability gap report and recommended next evidence source.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Do not invent or execute provider API calls without OpenAPI evidence.

## Fallback Behavior

- Stop after rendering the gap report.
