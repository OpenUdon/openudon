# IT Ops Incident Response Archive

## Goal

Create an incident ticket, alert the on-call Slack channel, and archive an incident timeline report to Google Drive.

## Inputs

- `service`: required string containing the affected service name.
- `severity`: required string containing the incident severity.
- `description`: required string containing the incident description.
- `timeline`: required string containing the initial incident timeline.
- `slackChannel`: required string containing the on-call Slack channel ID.
- `driveFolderId`: required string containing the Google Drive folder ID for incident reports.

## Outputs

- `archive_file`: Google Drive upload response body for the archived timeline.

## Data Flow

- Pass incident metadata into `create_jira_incident`.
- Bind `create_jira_incident.received_body.key` and incident metadata into `format_slack_alert`.
- Bind `format_slack_alert.received_body.text` into `post_slack_alert.text`.
- Bind Jira key and timeline into `render_timeline_report`.
- Bind `render_timeline_report.received_body.name` and `render_timeline_report.received_body.content` into `upload_timeline_report`.

## External Systems and OpenAPI

- Jira API: use `openapi/jira.yaml`.
- Slack API: use `openapi/slack.yaml`.
- Google Drive API: use `openapi/drive.yaml`.
- The fixture is inspired by a public n8n incident-response template and focuses on ticketing, alerting, and archival.

## Runtime Policy

- `openapi` and `http` are allowed for Jira, Slack, and Google Drive API requests.
- `fnct` is allowed for deterministic Slack alert and timeline report rendering.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `format_slack_alert`
  - Inputs: incident metadata and Jira issue key.
  - Outputs: Slack message text.
  - Side effects: none.
- `render_timeline_report`
  - Inputs: incident metadata, Jira issue key, and timeline.
  - Outputs: report file name and content.
  - Side effects: none.

## Credentials and Secrets

- Use credential binding name `jira_api_token` for Jira authentication.
- Use credential binding name `slack_bot_token` for Slack authentication.
- Use credential binding name `google_drive_oauth_token` for Google Drive authentication.
- Do not include tokens, production incident details, private channel IDs, or customer data in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate, validate, and assess artifacts only.
- Jira issue creation, Slack posting, and Drive upload are side-effectful and require human approval plus trusted-runner execution.
- Use sandbox Jira projects, Slack channels, and Drive folders for proof runs.

## Fallback Behavior

- Stop if Jira issue creation fails.
- Stop if Slack alert formatting fails.
- Stop if trusted execution approval is missing.
