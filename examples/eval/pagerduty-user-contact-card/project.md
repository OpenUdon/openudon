# PagerDuty User Contact Card

## Goal

Fetch one PagerDuty-style user profile and render a local contact card from the nested user response.

## Inputs

- `userId`: required string containing the user identifier.

## Outputs

- `contact_card`: rendered contact-card payload.

## Data Flow

- Pass `inputs.userId` to `get_user`.
- Bind `get_user.received_body.user.id`, `get_user.received_body.user.name`, and `get_user.received_body.user.email` into `render_contact_card`.

## External Systems and OpenAPI

- PagerDuty-like Users API: use `openapi/pagerduty.yaml`.
- The selected OpenAPI operation is `getUser`.

## Runtime Policy

- `openapi` and `http` are allowed for the Users API.
- `fnct` is allowed for local contact-card rendering.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `render_contact_card`
  - Inputs: userId, name, and email.
  - Outputs: contact-card payload.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required for this sandbox fixture.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- User lookup is read-only and may run against sandbox data after approval.

## Fallback Behavior

- Stop if the user cannot be fetched.
