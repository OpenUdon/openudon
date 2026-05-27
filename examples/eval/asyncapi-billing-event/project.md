# AsyncAPI Billing Event Eval

## Goal

Publish a reviewed invoice-created event to the package-local billing AsyncAPI source.

## Inputs

- `invoice_id`: required string for the invoice event payload.
- `customer_id`: required string for the invoice event payload.
- `trace_id`: required string for the event header.

## Outputs

- Event publish result from the trusted runtime.

## External Systems and API Sources

- Use `asyncapi/events.yaml` for billing event operations.
- OpenUdon validates and packages the AsyncAPI source-bound workflow, but AsyncAPI protocol execution belongs to the trusted runtime.

## Runtime Policy

- `openapi` and `http` are allowed for source-bound API/event operations.
- `fnct`, `cmd`, and `ssh` are not required.

## Data Flow

- Pass `invoice_id` and `customer_id` into the event payload under explicit `body.*` request mappings.
- Pass `trace_id` into the event header under an explicit `header.*` request mapping.

## Credentials and Secrets

- No credentials are required for this provider-free fixture.
- Do not place credential values in prompts, artifacts, or examples.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Use sandbox event topics or stubbed broker endpoints for proof runs before any production handoff.
- Real event delivery requires human approval and trusted-runner execution.

## Fallback Behavior

- Stop if the AsyncAPI operation is unavailable or required event details are missing.
