# Incomplete Brief Repair

## Goal

Render a local clarification checklist for an incomplete workflow brief.

## Inputs

- `brief`: required string containing the incomplete user brief.

## Outputs

- `clarification_checklist`: missing information checklist.

## Data Flow

- Pass `inputs.brief` to `render_clarification_checklist.brief`.

## External Systems and OpenAPI

OpenAPI: none required.

## Runtime Policy

- `fnct` is allowed for local checklist rendering.
- `openapi`, `http`, `cmd`, and `ssh` are not allowed.

## Function Contracts

- `render_clarification_checklist`
  - Inputs: incomplete brief.
  - Outputs: required clarifying questions and missing policy items.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required.
- Do not include secrets in the brief or checklist.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Do not synthesize external API calls from incomplete requirements.

## Fallback Behavior

- Stop after rendering the clarification checklist.
