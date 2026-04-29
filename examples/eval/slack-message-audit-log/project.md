# Slack Message Audit Log

## Goal

Post a sandbox Slack-style message and render a local audit log from the post response.

## Inputs

- `channel`: required string containing the channel identifier.
- `text`: required string containing the message text.

## Outputs

- `audit_log`: rendered audit log payload.

## Data Flow

- Pass `inputs.channel` and `inputs.text` to `post_message`.
- Bind `post_message.received_body.ok`, `post_message.received_body.channel`, and `post_message.received_body.ts` into `render_audit_log`.

## External Systems and OpenAPI

- Slack-like Chat API: use `openapi/slack.yaml`.
- The selected OpenAPI operation is `postMessage`.

## Runtime Policy

- `openapi` and `http` are allowed for the Chat API.
- `fnct` is allowed for local audit-log rendering.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `render_audit_log`
  - Inputs: ok, channel, and ts.
  - Outputs: audit log payload.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required for this sandbox fixture.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Posting a chat message is side-effectful and requires human approval plus trusted-runner execution.
- Use sandbox channels for proof runs.

## Fallback Behavior

- Stop if the message cannot be posted.
