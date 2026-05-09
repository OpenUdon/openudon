# n8n Slack Message Post

## Goal

Represent the n8n Slack message post workflow as a OpenUdon intent workflow that posts one message to one Slack channel.

## Inputs

- `channel`: required string containing the Slack channel ID or channel name.
- `text`: required string containing the message text.

## Outputs

- `message`: Slack `postMessage` response body returned by the post step.

## Data Flow

- Pass `inputs.channel` to `post_message.channel`.
- Pass `inputs.text` to `post_message.text`.
- Return `post_message.received_body` as `message`.

## External Systems and OpenAPI

- Slack API: use `openapi/slack.json`.
- The selected OpenAPI operation is `postMessage`.
- The fixture OpenAPI document is copied from `../w8m/reducibility/specs/slack.json` and enriched locally only for this eval slice.

## Runtime Policy

- `openapi` and `http` are allowed for the Slack API request.
- `fnct` is allowed only for non-side-effectful local shaping if future revisions need it.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- No `fnct` steps are part of this fixture.

## Credentials and Secrets

- Use credential binding name `slack_bot_token` for Slack authentication when this workflow is approved for execution.
- Do not include Slack bot tokens, channel secrets, workspace IDs, or production message content in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate, validate, and assess artifacts only.
- Do not run the Slack post workflow from this eval fixture.
- Posting a real Slack message is side-effectful and requires human approval plus trusted-runner execution.
- Use sandbox Slack workspaces or test endpoints for any future proof run.

## Fallback Behavior

- Stop if the Slack OpenAPI document is unavailable.
- Stop if `channel` or `text` is unavailable.
- Stop if trusted execution approval is missing.
