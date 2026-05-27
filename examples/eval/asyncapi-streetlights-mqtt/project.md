# AsyncAPI Streetlights MQTT Eval

## Goal

Publish a reviewed Streetlights MQTT dim-light command using a package-local AsyncAPI source.

## Inputs

- `streetlight_id`: required string for the streetlight topic parameter.
- `dim_percentage`: required number for the dim command payload percentage.
- `sent_at`: required string timestamp for the command payload.
- `app_header_value`: required string for the reviewed message header value.

## Outputs

- `dim_result`: dim command publish result from the trusted runtime.

## External Systems and OpenAPI

- Use `asyncapi/streetlights-mqtt.yml` for Streetlights MQTT event/message operations.
- No OpenAPI document is used for this fixture.
- OpenUdon validates and packages the AsyncAPI source-bound workflow, but MQTT protocol execution belongs to the trusted runtime.

## Runtime Policy

- `openapi` and `http` are allowed for source-bound API/event operations.
- `fnct`, `cmd`, and `ssh` are not required.

## Function Contracts

- No `fnct` helper contracts are required.

## Data Flow

- Pass `streetlight_id` into the event topic under an explicit `path.streetlightId` request mapping.
- Pass `dim_percentage` and `sent_at` into the event payload under explicit `body.*` request mappings.
- Pass `app_header_value` into the event header under an explicit `header.my-app-header` request mapping.

## Credentials and Secrets

- No credential values are stored in this provider-free fixture.
- Do not place broker credentials, API keys, OAuth tokens, or other secret values in prompts, artifacts, or examples.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Use sandbox or stubbed broker endpoints for proof runs before any production handoff.
- Real MQTT message delivery requires human approval and trusted-runner execution.

## Fallback Behavior

- Stop if the AsyncAPI operation is unavailable or required command details are missing.
