# Profile Boundary Manifest Eval

## Goal

Prepare a database export request manifest for an operator to review.

## Inputs

- `dataset`: dataset name to export.
- `since`: lower-bound timestamp for changed records.

## Outputs

- A JSON-style manifest describing the requested export.

## External Systems and OpenAPI

OpenAPI: none required

## Runtime Policy

- `fnct` is allowed for trusted local manifest rendering.
- `cmd`, `ssh`, and direct SQL/profile execution are not allowed.
- Do not emit `sql`, `smtp`, `llm`, `x-udon-*`, or other profile-specific runtime payloads from
  OpenUdon for this project.

## Data Flow

- `inputs.dataset` and `inputs.since` feed the manifest renderer.

## Function Contracts

- `render_export_manifest`
  - Inputs: `dataset`, `since`.
  - Outputs: `manifest`.
  - Side effects: none; renders a local review artifact only.

## Credentials and Secrets

- No credential bindings are required.
- Do not include database credentials, hostnames, or secret values.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- This fixture must not execute SQL, SSH, command, or profile runtime behavior.

## Fallback Behavior

- Stop if the export cannot be represented as a local manifest without direct profile execution.
