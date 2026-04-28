# Runtime Only Render Eval

## Goal

Render a local summary report using an approved function runtime.

## Inputs

- `summary`: required string.

## Outputs

- Rendered report body.

## External Systems and OpenAPI

OpenAPI: none required

## Runtime Policy

- `fnct` is allowed for trusted report rendering.
- `cmd` and `ssh` are not allowed.

## Data Flow

- Pass `inputs.summary` to `render_report.summary`.

## Function Contracts

- `render_report`
  - Inputs: summary.
  - Outputs: rendered report text.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate and validate artifacts only.

## Fallback Behavior

- Stop if no approved function runtime exists.
