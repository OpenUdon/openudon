# n8n PagerDuty User Get

## Goal

Represent the n8n PagerDuty user get workflow as a Ramen intent workflow that fetches one PagerDuty user.

## Inputs

- `userId`: required string containing the PagerDuty user ID.

## Outputs

- `user`: PagerDuty user response body returned by the get step.

## Data Flow

- Pass `inputs.userId` to `get_user.userId`.
- Return `get_user.received_body` as `user`.

## External Systems and OpenAPI

- PagerDuty API: use `openapi/pagerduty.json`.
- The selected OpenAPI operation is `getUser`.
- The OpenAPI document is copied from `../w8m/reducibility/specs/pagerduty.json`.

## Runtime Policy

- `openapi` and `http` are allowed for the PagerDuty API request.
- `fnct` is allowed only for non-side-effectful local shaping if future revisions need response cleanup.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- No `fnct` steps are part of this fixture.

## Credentials and Secrets

- Use credential binding name `pagerduty_oauth_token` for PagerDuty authentication when this workflow is approved for execution.
- Do not include OAuth tokens or production incident/user data in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate, validate, and assess artifacts only.
- Use sandbox PagerDuty accounts or test endpoints for any future proof run.
- Production execution requires human approval and trusted-runner execution.

## Fallback Behavior

- Stop if the PagerDuty OpenAPI document is unavailable.
- Stop if `userId` is unavailable.
- Stop if trusted execution approval is missing.
