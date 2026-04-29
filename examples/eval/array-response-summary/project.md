# Array Response Summary

## Goal

Fetch open alerts and render a local summary of affected services.

## Inputs

- `severity`: required string containing the minimum alert severity.

## Outputs

- `summary`: rendered alert summary.

## Data Flow

- Pass `inputs.severity` to `list_alerts.severity`.
- Bind `list_alerts.received_body.alerts` into `summarize_alerts.alerts`.

## External Systems and OpenAPI

- Alerts API: use `openapi/alerts.yaml`.
- The selected OpenAPI operation is `listAlerts`.

## Runtime Policy

- `openapi` and `http` are allowed for alert lookup.
- `fnct` is allowed for local summary rendering.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `summarize_alerts`
  - Inputs: alerts array.
  - Outputs: summary text and service counts.
  - Side effects: none.

## Credentials and Secrets

- Use credential binding name `alerts_bearer_token`.
- Do not include tokens or production alert data in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Use sandbox alerts for proof runs.

## Fallback Behavior

- Stop if alerts cannot be fetched.
