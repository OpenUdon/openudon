# Get Weather In Toronto

## Goal

- get weather in toronto, canada, and send me the report using Google gmail.

## Inputs

- `recipient_email`: required string.

## Outputs

- `result` from `render_weather_report.received_body`.

## Data Flow

- `retrieve_weather` depends on `geocode_openweathermap_location`.
- `retrieve_weather.lat` comes from `geocode_openweathermap_location.received_body[0].lat`.
- `retrieve_weather.lon` comes from `geocode_openweathermap_location.received_body[0].lon`.
- `render_weather_report` depends on `retrieve_weather`.
- `email_report` depends on `render_weather_report`.

## Function Contracts

- `render_weather_report`
  - Purpose: Render a reviewable local weather report from the weather response before Gmail delivery.
  - Function: `gmail.render_raw`.
  - Inputs: body_template, input, subject, to.
  - Outputs: received_body.
  - Side effects: none.

## External Systems and OpenAPI

- google-discovery/gmail-discovery-v1.json.
- openapi/openweathermap-one-call-3-overlay.json.

## Runtime Policy

- Allowed runtimes: `openapi`, `http`, `fnct`.
- `cmd` is not allowed unless explicitly approved here.
- `ssh` is not allowed unless explicitly approved here.

## Credentials and Secrets

- Name credential bindings only.
- Do not include secret values.
- Use credential binding `googleOAuth2`.
- Use credential binding `openWeatherAPIKey`.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Do not directly execute production workflows.
- Sandbox proof runs require review state `approved_for_sandbox`.
- Production execution requires review state `approved_for_production`.
- Side-effectful execution requires explicit approval, approved credential bindings, and a trusted runner.
- Trusted runner required for approved sandbox or production execution.

## Fallback Behavior

- Stop if required OpenAPI documents, runtime capabilities, or credential bindings are missing.
