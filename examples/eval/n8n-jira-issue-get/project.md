# n8n Jira Issue Get

## Goal

Represent the scanner-backed Jira issue get slice as a Ramen intent workflow that fetches one Jira issue.

## Inputs

- `issueKey`: required string containing the Jira issue key.

## Outputs

- `issue`: Jira issue response body returned by the get step.

## Data Flow

- Pass `inputs.issueKey` to `get_issue.issueKey`.
- Return `get_issue.received_body` as `issue`.

## External Systems and OpenAPI

- Jira API: use `openapi/jira.json`.
- The selected OpenAPI operation is `getIssue`.
- The OpenAPI document is copied from `../w8m/reducibility/specs/jira.json`.

## Runtime Policy

- `openapi` and `http` are allowed for the Jira API request.
- `fnct` is allowed only for non-side-effectful local shaping if future revisions need response cleanup.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- No `fnct` steps are part of this fixture.

## Credentials and Secrets

- Use credential binding name `jira_api_token` for Jira authentication when this workflow is approved for execution.
- Do not include Jira tokens, site URLs, or production issue data in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate, validate, and assess artifacts only.
- Use sandbox Jira projects or test endpoints for any future proof run.
- Production execution requires human approval and trusted-runner execution.

## Fallback Behavior

- Stop if the Jira OpenAPI document is unavailable.
- Stop if `issueKey` is unavailable.
- Stop if trusted execution approval is missing.
