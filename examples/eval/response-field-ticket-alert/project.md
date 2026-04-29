# Response Field Ticket Alert

## Goal

Fetch a ticket, extract nested assignee data, and send an internal alert.

## Inputs

- `ticketId`: required string containing the ticket ID.

## Outputs

- `send_status`: alert adapter send status.

## Data Flow

- Pass `inputs.ticketId` to `get_ticket.ticketId`.
- Bind `get_ticket.received_body.assignee.email` to `send_assignee_alert.to`.
- Bind `get_ticket.received_body.subject` to `send_assignee_alert.subject`.

## External Systems and OpenAPI

- Ticket API: use `openapi/tickets.yaml`.
- Alert delivery uses an approved `fnct` adapter.

## Runtime Policy

- `openapi` and `http` are allowed for ticket lookup.
- `fnct` is allowed for the approved alert adapter.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `send_assignee_alert`
  - Inputs: to, subject, and body.
  - Outputs: send status.
  - Side effects: sends only through the approved trusted adapter.

## Credentials and Secrets

- Use credential binding name `tickets_bearer_token`.
- Do not include tokens or production ticket content in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Alert sending is side-effectful and requires human approval plus trusted-runner execution.
- Use sandbox alert endpoints for proof runs.

## Fallback Behavior

- Stop if the ticket or nested assignee email is unavailable.
- Stop if trusted execution approval is missing.
