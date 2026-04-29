# API Nested User Create

## Goal

Create one sandbox user account with nested profile metadata.

## Inputs

- `email`: required string containing the user email.
- `displayName`: required string containing the display name.
- `department`: required string containing the profile department.

## Outputs

- `user`: created user response body.

## Data Flow

- Pass `inputs.email` to `create_user.email`.
- Pass `inputs.displayName` to `create_user.displayName`.
- Pass `inputs.department` to `create_user.profile`.

## External Systems and OpenAPI

- User Admin API: use `openapi/users.yaml`.
- The selected OpenAPI operation is `createUser`.

## Runtime Policy

- `openapi` and `http` are allowed for the User Admin API.
- `fnct` is allowed only for non-side-effectful local shaping if future revisions need it.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- No `fnct` steps are part of this fixture.

## Credentials and Secrets

- Use credential binding name `user_admin_token` for User Admin API authentication.
- Do not include tokens or production user data in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- User creation is side-effectful and requires human approval plus trusted-runner execution.
- Use sandbox user directories for proof runs.

## Fallback Behavior

- Stop if required user fields are unavailable.
- Stop if trusted execution approval is missing.
