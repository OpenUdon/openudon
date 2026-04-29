# Webhook Validation Function

## Goal

Validate an incoming webhook payload and render an accepted payload summary without external API calls.

## Inputs

- `payload`: required object containing the webhook body.
- `signature`: required string containing the received signature value.

## Outputs

- `validation`: validation result and normalized summary.

## Data Flow

- Pass `inputs.payload` and `inputs.signature` to `validate_webhook`.

## External Systems and OpenAPI

OpenAPI: none required.

## Runtime Policy

- `fnct` is allowed for deterministic validation logic.
- `openapi`, `http`, `cmd`, and `ssh` are not allowed.

## Function Contracts

- `validate_webhook`
  - Inputs: payload and signature.
  - Outputs: validation status and normalized summary.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required.
- Do not include signing secrets in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Validation is local and read-only.

## Fallback Behavior

- Stop if the payload or signature is unavailable.
