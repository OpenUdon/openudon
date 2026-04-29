# n8n HubSpot Deal List

## Goal

Represent the n8n HubSpot deal get-many workflow as a Ramen intent workflow that lists deals.

## Inputs

- `limit`: required integer containing the maximum number of deals to request.

## Outputs

- `deals`: HubSpot deal list response body returned by the list step.

## Data Flow

- Pass `inputs.limit` to `list_deals.limit`.
- Return `list_deals.received_body` as `deals`.

## External Systems and OpenAPI

- HubSpot API: use `openapi/hubspot.json`.
- The selected OpenAPI operation is `listDeals`.
- The OpenAPI document is copied from `../w8m/reducibility/specs/hubspot.json`.

## Runtime Policy

- `openapi` and `http` are allowed for the HubSpot API request.
- `fnct` is allowed only for non-side-effectful local shaping if future revisions need pagination cleanup.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- No `fnct` steps are part of this fixture.

## Credentials and Secrets

- Use credential binding name `hubspot_private_app_token` for HubSpot authentication when this workflow is approved for execution.
- Do not include HubSpot tokens or production CRM data in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate, validate, and assess artifacts only.
- Use sandbox HubSpot accounts or test endpoints for any future proof run.
- Production execution requires human approval and trusted-runner execution.

## Fallback Behavior

- Stop if the HubSpot OpenAPI document is unavailable.
- Stop if `limit` is unavailable.
- Stop if trusted execution approval is missing.
