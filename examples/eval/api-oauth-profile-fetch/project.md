# API OAuth Profile Fetch

## Goal

Fetch one employee profile using a bearer credential binding.

## Inputs

- `employeeId`: required string containing the employee ID.

## Outputs

- `profile`: employee profile response body.

## Data Flow

- Pass `inputs.employeeId` to `get_profile.employeeId`.
- Pass credential binding `directory_bearer_token` to `get_profile.Authorization`.

## External Systems and OpenAPI

- Directory API: use `openapi/directory.yaml`.
- The selected OpenAPI operation is `getEmployeeProfile`.

## Runtime Policy

- `openapi` and `http` are allowed for the Directory API.
- `fnct` is allowed only for non-side-effectful local shaping if future revisions need it.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- No `fnct` steps are part of this fixture.

## Credentials and Secrets

- Use credential binding name `directory_bearer_token`.
- Do not include bearer tokens or production profile data in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Use sandbox directory data for proof runs.

## Fallback Behavior

- Stop if `employeeId` is unavailable.
- Stop if the bearer credential binding is missing.
