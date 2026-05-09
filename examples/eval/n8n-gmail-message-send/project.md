# n8n Gmail Message Send

## Goal

Represent the n8n Gmail message send workflow as a OpenUdon intent workflow that sends one email message.

## Inputs

- `sendTo`: required string containing the recipient email address.
- `subject`: required string containing the email subject.
- `message`: required string containing the email body.

## Outputs

- `sent_message`: Gmail send response body returned by the send step.

## Data Flow

- Pass `inputs.sendTo` to `send_message.sendTo`.
- Pass `inputs.subject` to `send_message.subject`.
- Pass `inputs.message` to `send_message.message`.
- Return `send_message.received_body` as `sent_message`.

## External Systems and OpenAPI

- Gmail API: use `openapi/gmail.json`.
- The selected OpenAPI operation is `sendMessage`.
- The OpenAPI document is copied from `../w8m/reducibility/specs/gmail.json`.

## Runtime Policy

- `openapi` and `http` are allowed for the Gmail API request.
- `fnct` is allowed only for non-side-effectful local shaping if future revisions need raw MIME assembly.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- No `fnct` steps are part of this fixture.

## Credentials and Secrets

- Use credential binding name `gmail_oauth_token` for Gmail authentication when this workflow is approved for execution.
- Do not include OAuth tokens, real recipients, or production message content in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate, validate, and assess artifacts only.
- Do not run the Gmail send workflow from this eval fixture.
- Sending real email is side-effectful and requires human approval plus trusted-runner execution.
- Use sandbox Gmail accounts or test endpoints for any future proof run.

## Fallback Behavior

- Stop if the Gmail OpenAPI document is unavailable.
- Stop if `sendTo`, `subject`, or `message` is unavailable.
- Stop if trusted execution approval is missing.
