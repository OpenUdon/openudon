# M28 Support Ticket Slack Status

## Goal

M28 sample: convert one support message into a ticket and post a Slack status confirmation.

## Inputs

- `channel`: required string containing the Slack channel ID.
- `messageTs`: required string containing the Slack message timestamp.
- `projectKey`: required string containing the Jira project key.

## Outputs

- `issue_key`: created Jira issue key.

## Data Flow

- Pass `inputs.channel` and `inputs.messageTs` to `get_slack_message`.
- Bind `get_slack_message.received_body.message.text` into `parse_issue_report.text`.
- Bind parsed title, description, priority, and issue type into `create_jira_issue`.
- Bind `create_jira_issue.received_body.key` and `create_jira_issue.received_body.self` into `post_slack_confirmation`.

## External Systems and OpenAPI

- Slack API: use `openapi/slack.yaml`.
- Jira API: use `openapi/jira.yaml`.
- The fixture is inspired by a public Slack-to-Jira n8n issue-intake template and omits attachment download/upload from the first OpenUdon eval slice.

## Runtime Policy

- `openapi` and `http` are allowed for Slack and Jira API requests.
- `fnct` is allowed for deterministic Slack message parsing.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `parse_issue_report`
  - Inputs: Slack message text.
  - Outputs: title, description, priority, and issueType.
  - Side effects: none.

## Credentials and Secrets

- Use credential binding name `slack_bot_token` for Slack authentication.
- Use credential binding name `jira_api_token` for Jira authentication.
- Do not include bot tokens, Jira tokens, private channel IDs, or production issue text in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate, validate, and assess artifacts only.
- Jira issue creation and Slack posting are side-effectful and require human approval plus trusted-runner execution.
- Use sandbox Slack channels and Jira projects for proof runs.

## Fallback Behavior

- Stop if the Slack message cannot be fetched.
- Stop if the Slack message cannot be parsed into required Jira fields.
- Stop if trusted execution approval is missing.
