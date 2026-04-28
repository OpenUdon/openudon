# Support Email Example

## Goal

When a support ticket is created, fetch the ticket details from the Support API and send a
confirmation email to the requester through SMTP.

## Inputs

- `ticket_id`: required string from the incoming `ticket_created` event.

## Outputs

- Confirmation email send request.
- Trusted runtime result or validation report.

## External Systems and OpenAPI

- Support API: use the OpenAPI document under `openapi/`.
- SMTP is not currently a first-class udon runtime. Model email delivery through an approved
  `fnct` adapter or stop if no approved adapter exists.

## Runtime Policy

- `openapi`/`http` allowed for Support API lookup.
- `fnct` allowed only for approved email adapter glue.
- `cmd` and `ssh` are not allowed.

## Data Flow

- Pass `get_ticket.received_body.requester.email` to the email adapter `to` field.
- Pass `get_ticket.received_body.subject` to the email adapter `subject` field.
- Pass `get_ticket.received_body.summary` to the email adapter body renderer.

## Function Contracts

- `send_confirmation_email`
  - Inputs: requester email, ticket subject, ticket summary.
  - Outputs: send status and provider message ID when available.
  - Side effects: sends email only through the approved trusted runtime path.

## Credentials and Secrets

- Use credential binding names only.
- Do not include SMTP, Support API, or LLM credential values in generated artifacts.

## Intended Workflow

1. Receive or dispatch a `ticket_created` event containing a ticket ID.
2. Call the Support API using its OpenAPI operation to fetch ticket details.
3. Extract the requester's email address, ticket subject, and summary from the API response.
4. Send an SMTP email through an approved runtime adapter if one is available.
5. Persist or report the execution result through the trusted `udon` runtime path.

## Artifact Expectations

- `openapi/` should contain the Support API OpenAPI description.
- `workflows/` should contain the generated UWS workflow.
- `expected/` should contain expected result notes, fixtures, or review evidence.

## Safety Boundary

Agents may generate and validate the workflow. Sending real email requires an approved runtime
execution path with trusted SMTP credentials.
Use sandbox email endpoints for proof runs before any production email handoff, and require human
approval plus trusted-runner execution for real delivery.

## Fallback Behavior

- Stop if the Support API OpenAPI document is missing.
- Stop if no approved SMTP adapter/runtime exists.
