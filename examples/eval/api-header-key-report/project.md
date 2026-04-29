# API Header Key Report

## Goal

Fetch a compliance report using an API key carried in a request header.

## Inputs

- `reportId`: required string containing the report ID.

## Outputs

- `report`: compliance report response body.

## Data Flow

- Pass `inputs.reportId` to `get_report.reportId`.
- Pass credential binding `compliance_api_key` to the required `api_key_auth` security field.

## External Systems and OpenAPI

- Compliance API: use `openapi/compliance.yaml`.
- The selected OpenAPI operation is `getComplianceReport`.

## Runtime Policy

- `openapi` and `http` are allowed for the Compliance API.
- `fnct` is allowed only for non-side-effectful local shaping if future revisions need it.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- No `fnct` steps are part of this fixture.

## Credentials and Secrets

- Use credential binding name `compliance_api_key` for the API-key header.
- Do not include API keys or production compliance data in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Use sandbox compliance reports for proof runs.

## Fallback Behavior

- Stop if `reportId` is unavailable.
- Stop if the API key binding is missing.
