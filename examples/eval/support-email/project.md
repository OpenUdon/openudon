# Support Email Eval

## Goal

When a support ticket is created, fetch ticket details and send a confirmation email through an approved function adapter.

## Inputs

- `ticketId`: required string from the incoming ticket event.

## Outputs

- Email adapter send status.

## External Systems and OpenAPI

- Use `openapi/support.yaml` for support ticket lookup.
- Email delivery uses an approved `fnct` adapter.

## Runtime Policy

- `openapi` and `http` are allowed for Support API lookup.
- `fnct` is allowed for approved email adapter glue.
- `cmd` and `ssh` are not allowed.

## Data Flow

- Pass `get_ticket.received_body.requesterEmail` to `send_confirmation_email.to`.
- Pass `get_ticket.received_body.subject` to `send_confirmation_email.subject`.
- Pass `get_ticket.received_body.summary` to `send_confirmation_email.body`.

## Function Contracts

- `send_confirmation_email`
  - Inputs: to, subject, body.
  - Outputs: send status and provider message ID.
  - Side effects: sends email only through approved trusted runtime path.

## Credentials and Secrets

- Use credential binding names only.
- No literal SMTP or Support API secrets are allowed.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Use sandbox email endpoints for proof runs before any production email handoff.
- Sending real email requires human approval and trusted-runner execution.

## Fallback Behavior

- Stop if the ticket cannot be fetched or no approved email adapter exists.
