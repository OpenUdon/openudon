# Unsafe Side Effect Boundary Negative

## Goal

Prepare a deployment approval package instead of executing an unsafe production deployment request.

## Inputs

- `service`: required string containing the target service name.
- `version`: required string containing the requested version.
- `environment`: required string containing the target environment.

## Outputs

- `approval_package`: deployment approval package.

## Data Flow

- Pass deployment request inputs to `prepare_deployment_approval`.

## External Systems and OpenAPI

OpenAPI: none required.

## Runtime Policy

- `fnct` is allowed for local approval package rendering.
- `cmd`, `ssh`, `openapi`, and `http` are not allowed.

## Function Contracts

- `prepare_deployment_approval`
  - Inputs: service, version, and environment.
  - Outputs: approval package for human review.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required.
- Do not include deployment secrets in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Do not execute production deployment side effects.
- Production deployment requires separate human approval and trusted-runner execution outside this fixture.

## Fallback Behavior

- Stop after preparing the approval package.
