# Retry Idempotent Webhook Send

## Goal

Send one idempotent webhook notification with explicit timeout and idempotency metadata.

## Inputs

- `eventId`: required string containing the idempotency key.
- `payload`: required object containing the webhook payload.

## Outputs

- `delivery`: webhook delivery response body.

## Data Flow

- Pass `inputs.eventId` to `send_webhook.idempotencyKey`.
- Pass `inputs.payload` to `send_webhook.payload`.

## External Systems and OpenAPI

- Webhook Delivery API: use `openapi/webhooks.yaml`.
- The selected OpenAPI operation is `sendWebhook`.

## Runtime Policy

- `openapi` and `http` are allowed for webhook delivery.
- `fnct` is allowed only for non-side-effectful local shaping if future revisions need it.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- No `fnct` steps are part of this fixture.

## Credentials and Secrets

- Use credential binding name `webhook_bearer_token`.
- Do not include tokens or production payloads in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Webhook delivery is side-effectful and requires human approval plus trusted-runner execution.
- The send operation is idempotent and safe to retry with a bounded retry limit.
- Use sandbox webhook endpoints for proof runs.

## Fallback Behavior

- Stop if `eventId` or `payload` is unavailable.
- Stop if trusted execution approval is missing.
