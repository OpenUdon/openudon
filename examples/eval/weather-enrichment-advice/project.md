# Weather Enrichment Advice

## Goal

Fetch current weather for one city and render local advice from the weather response.

## Inputs

- `city`: required string containing the city name.

## Outputs

- `advice`: rendered weather advice payload.

## Data Flow

- Pass `inputs.city` to `get_weather`.
- Bind `get_weather.received_body.city`, `get_weather.received_body.tempC`, and `get_weather.received_body.condition` into `render_weather_advice`.

## External Systems and OpenAPI

- Weather API: use `openapi/weather.yaml`.
- The selected OpenAPI operation is `getWeather`.

## Runtime Policy

- `openapi` and `http` are allowed for the Weather API.
- `fnct` is allowed for local advice rendering.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `render_weather_advice`
  - Inputs: city, tempC, and condition.
  - Outputs: weather advice payload.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required for this sandbox fixture.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Weather lookup is read-only and may run against sandbox data after approval.

## Fallback Behavior

- Stop if weather cannot be fetched.
