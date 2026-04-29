# User Create Welcome Message

## Goal

Create a sandbox user account and render a welcome message from the created user response.

## Inputs

- `email`: required string containing the user email.
- `displayName`: required string containing the display name.

## Outputs

- `welcome_message`: rendered welcome message.

## Data Flow

- Pass `inputs.email` and `inputs.displayName` to `create_user`.
- Bind `create_user.received_body.id`, `create_user.received_body.email`, and `create_user.received_body.displayName` into `render_welcome_message`.

## External Systems and OpenAPI

- User Admin API: use `openapi/users.yaml`.
- The selected OpenAPI operation is `createUser`.

## Runtime Policy

- `openapi` and `http` are allowed for the User Admin API.
- `fnct` is allowed for local welcome-message rendering.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `render_welcome_message`
  - Inputs: userId, email, and displayName.
  - Outputs: welcome message payload.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required for this sandbox fixture.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- User creation is side-effectful and requires human approval plus trusted-runner execution.
- Use sandbox user directories for proof runs.

## Fallback Behavior

- Stop if the user cannot be created.
