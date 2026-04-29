# n8n Airtable Record Get

## Goal

Represent the n8n Airtable record get workflow as a Ramen intent workflow that fetches one Airtable record.

## Inputs

- `baseId`: required string containing the Airtable base ID.
- `tableId`: required string containing the Airtable table ID or name.
- `recordId`: required string containing the Airtable record ID.

## Outputs

- `record`: Airtable record response body returned by the get step.

## Data Flow

- Pass `inputs.baseId` to `get_record.baseId`.
- Pass `inputs.tableId` to `get_record.tableId`.
- Pass `inputs.recordId` to `get_record.id`.
- Return `get_record.received_body` as `record`.

## External Systems and OpenAPI

- Airtable API: use `openapi/airtable.json`.
- The selected OpenAPI operation is `getAirtableRecord`.
- The OpenAPI document is copied from `../w8m/reducibility/specs/airtable.json`.

## Runtime Policy

- `openapi` and `http` are allowed for the Airtable API request.
- `fnct` is allowed only for non-side-effectful local shaping if future revisions need it.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- No `fnct` steps are part of this fixture.

## Credentials and Secrets

- Use credential binding name `airtable_access_token` for Airtable authentication when this workflow is approved for execution.
- Do not include Airtable tokens or production base data in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate, validate, and assess artifacts only.
- Use sandbox Airtable bases or test endpoints for any future proof run.
- Production execution requires human approval and trusted-runner execution.

## Fallback Behavior

- Stop if the Airtable OpenAPI document is unavailable.
- Stop if `baseId`, `tableId`, or `recordId` is unavailable.
- Stop if trusted execution approval is missing.
