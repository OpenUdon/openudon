# M28 Gmail Audit Receipt

## Goal

Send a sandbox email message and render a local audit receipt from the provider response.

## Inputs

- `to`: required string containing the recipient email.
- `subject`: required string containing the message subject.
- `message`: required string containing the message body.

## Outputs

- `audit_receipt`: rendered audit receipt payload.

## Data Flow

- Pass `inputs.to`, `inputs.subject`, and `inputs.message` to `send_message`.
- Bind `send_message.received_body.id` and `send_message.received_body.threadId` into `render_audit_receipt`.

## External Systems and OpenAPI

- Gmail-like Messages API: use `openapi/gmail.yaml`.
- The selected OpenAPI operation is `sendMessage`.

## Runtime Policy

- `openapi` and `http` are allowed for the Messages API.
- `fnct` is allowed for local audit receipt rendering.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `render_audit_receipt`
  - Inputs: messageId and threadId.
  - Outputs: audit receipt payload.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required for this sandbox fixture.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Sending email is side-effectful and requires human approval plus trusted-runner execution.
- Use sandbox recipients for proof runs.

## Fallback Behavior

- Stop if the email cannot be sent.
