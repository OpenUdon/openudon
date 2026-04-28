# Support Priority Routing

## Goal

Fetch a support ticket, classify its priority, and prepare one internal routing result.

## Inputs

- `ticketId`: required string from the incoming support event.

## Outputs

- `routing_result`: selected internal handling result.

## External Systems and OpenAPI

- Support API: use `openapi/support.yaml`.
- OpenAPI is required for ticket lookup.

## Data Flow

- Pass `inputs.ticketId` to `get_ticket.ticketId`.
- Pass `get_ticket.received_body` to `classify_priority.ticket`.
- Pass `get_ticket.received_body` and `classify_priority.received_body` to `prepare_routing_result`.
- The `prepare_routing_result` function chooses the urgent or standard internal handling path.

## Function Contracts

- `classify_priority`
  - Inputs: ticket body from `get_ticket`.
  - Outputs: priority label and rationale.
  - Side effects: none.
- `prepare_routing_result`
  - Inputs: ticket body and priority classification.
  - Outputs: selected internal handling result.
  - Side effects: none.

## Runtime Policy

- `openapi` and `http` are allowed for the Support API.
- `fnct` is allowed for classification and internal response preparation.
- `cmd` and `ssh` are not allowed.

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Do not send an outbound customer message.

## Fallback Behavior

- Stop if the ticket cannot be fetched.
- Stop if no approved function runtime exists for classification or routing preparation.
