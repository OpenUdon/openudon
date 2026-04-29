# n8n OpenWeatherMap Current Weather

## Goal

Represent the n8n OpenWeatherMap current weather workflow as a Ramen intent workflow that fetches current weather for one city.

## Inputs

- `cityName`: required string containing the city and country query.
- `language`: required string containing the response language code.

## Outputs

- `weather`: OpenWeatherMap current weather response body returned by the weather step.

## Data Flow

- Pass `inputs.cityName` to `get_current_weather.q`.
- Pass `inputs.language` to `get_current_weather.lang`.
- Return `get_current_weather.received_body` as `weather`.

## External Systems and OpenAPI

- OpenWeatherMap API: use `openapi/openweathermap.json`.
- The selected OpenAPI operation is `getOpenWeatherMapCurrentWeather`.
- The OpenAPI document is copied from `../w8m/reducibility/specs/openweathermap.json`.

## Runtime Policy

- `openapi` and `http` are allowed for the OpenWeatherMap API request.
- `fnct` is allowed only for non-side-effectful local shaping if future revisions need response mapping.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- No `fnct` steps are part of this fixture.

## Credentials and Secrets

- Use credential binding name `openweathermap_appid` for OpenWeatherMap authentication when this workflow is approved for execution.
- Do not include API keys or production request data in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate, validate, and assess artifacts only.
- Use sandbox/test endpoints for any future proof run.
- Production execution requires human approval and trusted-runner execution.

## Fallback Behavior

- Stop if the OpenWeatherMap OpenAPI document is unavailable.
- Stop if `cityName` or `language` is unavailable.
- Stop if trusted execution approval is missing.
