# Command Allowed Deploy Eval

## Goal

Run the approved deployment status command and return its output.

## Inputs

- No inputs are required.

## Outputs

- Deployment status text.

## External Systems and OpenAPI

OpenAPI: none required

## Runtime Policy

- `cmd` is explicitly allowed for the sandbox deployment status command.
- `ssh` is not allowed.
- `fnct` is allowed for trusted adapters.

## Data Flow

- No cross-step data flow is required.

## Function Contracts

- No function steps are expected.

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate and validate artifacts only.

## Fallback Behavior

- Stop if the command runtime is unavailable.
