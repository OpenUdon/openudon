# M28 Airtable Normalization

## Goal

Fetch one Airtable-style record and normalize the response fields into a local payload.

## Inputs

- `baseId`: required string containing the base identifier.
- `tableId`: required string containing the table identifier.
- `recordId`: required string containing the record identifier.

## Outputs

- `normalized_record`: normalized local record payload.

## Data Flow

- Pass `inputs.baseId`, `inputs.tableId`, and `inputs.recordId` to `get_record`.
- Bind `get_record.received_body.id`, `get_record.received_body.fields`, and `get_record.received_body.createdTime` into `normalize_record`.

## External Systems and OpenAPI

- Airtable-like Records API: use `openapi/airtable.yaml`.
- The selected OpenAPI operation is `getAirtableRecord`.

## Runtime Policy

- `openapi` and `http` are allowed for the Records API.
- `fnct` is allowed for local normalization.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `normalize_record`
  - Inputs: recordId, fields, and createdTime.
  - Outputs: normalized record payload.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required for this sandbox fixture.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Record reads are read-only and may run against sandbox data after approval.

## Fallback Behavior

- Stop if the record cannot be fetched.
