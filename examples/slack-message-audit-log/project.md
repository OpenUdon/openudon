# Slack Message Audit Log

## Goal

- Post a sandbox chat message and render a local audit log from the response.

## Inputs

- `channel`: required string.
- `text`: required string.

## Outputs

- `audit_log` from `render_audit_log.received_body`.

## Data Flow

- `render_audit_log` depends on `post_message`.
- `render_audit_log.channel` comes from `post_message.received_body.channel`.
- `render_audit_log.ok` comes from `post_message.received_body.ok`.
- `render_audit_log.ts` comes from `post_message.received_body.ts`.

## Function Contracts

- `render_audit_log`
  - Purpose: Render a local audit log from the post response.
  - Inputs: channel, ok, ts.
  - Outputs: received_body.
  - Side effects: declared by approved function adapter; execute only through trusted runner approval.

## External Systems and OpenAPI

- openapi/slack.yaml.

## Runtime Policy

- Allowed runtimes: `openapi`, `http`, `fnct`.
- `cmd` is not allowed unless explicitly approved here.
- `ssh` is not allowed unless explicitly approved here.

## Credentials and Secrets

- Name credential bindings only.
- Do not include secret values.
- Use credential binding `fixture.`.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Do not directly execute production workflows.
- Sandbox proof runs require review state `approved_for_sandbox`.
- Production execution requires review state `approved_for_production`.
- Side-effectful execution requires explicit approval, approved credential bindings, and a trusted runner.
- Trusted runner required for approved sandbox or production execution.
- Use sandbox channels for proof runs.

## Fallback Behavior

- Stop if the message cannot be posted.
