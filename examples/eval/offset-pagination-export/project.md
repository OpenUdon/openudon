# Offset Pagination Export

## Goal

Fetch two offset-paginated pages of audit records and merge them into one export payload.

## Inputs

- No inputs are required.

## Outputs

- `export_payload`: merged audit record export.

## Data Flow

- Fetch page one with `offset` 0 and `limit` 100.
- Fetch page two with `offset` 100 and `limit` 100.
- Bind both page record arrays into `merge_audit_pages`.

## External Systems and OpenAPI

- Audit API: use `openapi/audit.yaml`.
- The selected OpenAPI operation is `listAuditRecords`.

## Runtime Policy

- `openapi` and `http` are allowed for the Audit API.
- `fnct` is allowed for local merge rendering.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `merge_audit_pages`
  - Inputs: page_1 and page_2 record arrays.
  - Outputs: merged export payload.
  - Side effects: none.

## Credentials and Secrets

- Use credential binding name `audit_bearer_token`.
- Do not include tokens or production audit data in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Use sandbox audit records for proof runs.

## Fallback Behavior

- Stop if either page cannot be fetched.
