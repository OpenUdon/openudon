# Project Name

## Goal

Describe the workflow outcome in business terms.

## Inputs

- List trigger payloads, user-provided values, files, or environment-provided bindings.

## Outputs

- List generated artifacts, API writes, files, notifications, or reports.

## Data Flow

- List important mappings, for example: `get_coordinates.body[0].lat -> get_weather.lat`.
- If the user-level goal may require hidden API steps, describe the business action plainly.

## Function Contracts

- `function_name`
  - Inputs: list required fields or prior-step outputs.
  - Outputs: list returned fields.
  - Side effects: none, or describe approved side effects.

## External Systems and OpenAPI

- List APIs/services involved and the OpenAPI files or URLs to use.
- If no API/OpenAPI integration is needed, write: `OpenAPI: none required`.

## Runtime Policy

- Allowed runtimes: `openapi`, `http`, `fnct`.
- `cmd` is not allowed unless explicitly approved here.
- `ssh` is not allowed unless explicitly approved here.

## Credentials and Secrets

- Name credential bindings only.
- Do not include secret values.

## Safety and Approval Boundary

- Describe what Ramen may generate, validate, or execute.
- Side-effectful execution requires explicit approval, sandbox/test proof runs, and a trusted runner.
- Ramen synthesis must not directly execute production workflows.

## Fallback Behavior

- Stop if required OpenAPI documents, runtime capabilities, or credential bindings are missing.
