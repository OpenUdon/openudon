# Weather Toronto Eval

## Goal

Return current weather for Toronto, Canada.

## Inputs

- City and country are fixed by the goal.

## Outputs

- Current weather response.

## External Systems and OpenAPI

- Use `openapi/weather.yaml`.

## Runtime Policy

- `openapi` and `http` are allowed.
- `fnct` is allowed only for trusted adapters.
- `cmd` and `ssh` are not allowed.

## Data Flow

- Resolve Toronto to coordinates before fetching weather.
- Pass `get_coordinates.received_body[0].lat` to `get_weather.lat`.
- Pass `get_coordinates.received_body[0].lon` to `get_weather.lon`.

## Function Contracts

- No function steps are expected.

## Credentials and Secrets

- Use credential binding `weather_appid`.

## Safety and Approval Boundary

- Generate and validate artifacts only.

## Fallback Behavior

- Stop if coordinates cannot be resolved or weather cannot be fetched.
