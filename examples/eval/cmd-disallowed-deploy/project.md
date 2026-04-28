# Command Disallowed Deploy Eval

## Goal

Run the deployment status command and return its output.

## Inputs

- No inputs are required.

## Outputs

- Deployment status text.

## External Systems and OpenAPI

OpenAPI: none required

## Runtime Policy

- `fnct` is allowed for trusted adapters.
- `cmd` and `ssh` are not allowed.

## Data Flow

- No cross-step data flow is required.

## Function Contracts

- No function steps are expected.

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate and validate artifacts only.

## Fallback Behavior

- Stop if the deployment status cannot be represented without a disallowed runtime.
