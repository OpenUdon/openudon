# Compliance Report Summary

## Goal

Fetch a compliance report and render a local status summary from the API response.

## Inputs

- `reportId`: required string containing the report ID.

## Outputs

- `summary`: rendered compliance report summary.

## Data Flow

- Pass `inputs.reportId` to `get_report`.
- Bind `get_report.received_body.id`, `get_report.received_body.status`, and `get_report.received_body.ownerEmail` into `render_report_summary`.

## External Systems and OpenAPI

- Compliance API: use `openapi/compliance.yaml`.
- The selected OpenAPI operation is `getComplianceReport`.

## Runtime Policy

- `openapi` and `http` are allowed for the Compliance API.
- `fnct` is allowed for local report summary rendering.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `render_report_summary`
  - Inputs: reportId, status, and ownerEmail.
  - Outputs: summary payload.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required for this sandbox fixture.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Use sandbox report data for proof runs.

## Fallback Behavior

- Stop if the report cannot be fetched.
