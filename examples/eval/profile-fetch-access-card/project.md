# Profile Fetch Access Card

## Goal

Fetch an employee profile and render an access review card from the returned profile fields.

## Inputs

- `employeeId`: required string containing the employee ID.

## Outputs

- `access_card`: rendered access review card.

## Data Flow

- Pass `inputs.employeeId` to `get_profile`.
- Bind `get_profile.received_body.email`, `get_profile.received_body.managerEmail`, and `get_profile.received_body.department` into `render_access_card`.

## External Systems and OpenAPI

- Directory API: use `openapi/directory.yaml`.
- The selected OpenAPI operation is `getEmployeeProfile`.

## Runtime Policy

- `openapi` and `http` are allowed for the Directory API.
- `fnct` is allowed for local access-card rendering.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `render_access_card`
  - Inputs: email, managerEmail, and department.
  - Outputs: access review card.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required for this sandbox fixture.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Use sandbox directory data for proof runs.

## Fallback Behavior

- Stop if the profile cannot be fetched.
