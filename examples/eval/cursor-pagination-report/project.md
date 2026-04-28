# Cursor Pagination Report

## Goal

Fetch the first two pages of audit events using cursor pagination and render a local report.

## Inputs

- No user inputs are required.

## Outputs

- `report`: rendered two-page audit event report.

## External Systems and OpenAPI

- Audit Events API: use `openapi/audit-events.yaml`.
- OpenAPI is required for audit event retrieval.

## Data Flow

- Fetch the first page with literal `limit = 100`.
- Pass `list_events_first_page.received_body.page.nextCursor` to `list_events_second_page.cursor`.
- Fetch the second page with literal `limit = 100`.
- Pass both pages' `received_body.events` arrays to `render_audit_report`.

## Function Contracts

- `render_audit_report`
  - Inputs: event arrays from page 1 and page 2.
  - Outputs: rendered audit report.
  - Side effects: none.

## Runtime Policy

- `openapi` and `http` are allowed for the Audit Events API.
- `fnct` is allowed for local report rendering.
- `cmd` and `ssh` are not allowed.

## Credentials and Secrets

- Use credential binding `audit_events_bearer_token`.
- Never include bearer token values in generated artifacts.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Do not write files or send the report anywhere.

## Fallback Behavior

- Stop if either audit event page cannot be fetched.
- Stop if no approved report-rendering function runtime exists.
