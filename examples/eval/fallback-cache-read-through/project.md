# Fallback Cache Read Through

## Goal

Fetch a feature flag from the primary API and prepare a cached fallback value when the primary value is unavailable.

## Inputs

- `flagKey`: required string containing the feature flag key.

## Outputs

- `flag_result`: selected primary-or-cache flag result.

## Data Flow

- Pass `inputs.flagKey` to `get_primary_flag.flagKey`.
- Bind `get_primary_flag.received_body` and `inputs.flagKey` into `prepare_cached_fallback`.
- Bind both primary and fallback payloads into `select_flag_result`.

## External Systems and OpenAPI

- Feature Flag API: use `openapi/flags.yaml`.
- Fallback cache lookup is represented as approved local `fnct` behavior.

## Runtime Policy

- `openapi` and `http` are allowed for the primary feature flag API.
- `fnct` is allowed for cache fallback preparation and result selection.
- `cmd` and `ssh` are not allowed.

## Function Contracts

- `prepare_cached_fallback`
  - Inputs: flagKey and primary response.
  - Outputs: cached fallback candidate.
  - Side effects: none.
- `select_flag_result`
  - Inputs: primary response and cached fallback candidate.
  - Outputs: selected result.
  - Side effects: none.

## Credentials and Secrets

- Use credential binding name `flags_bearer_token`.
- Do not include tokens or production flag values in prompts, examples, or artifacts.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Use sandbox flag data for proof runs.

## Fallback Behavior

- Use the cached fallback candidate only when the primary response is unavailable or unusable.
- Stop if neither primary nor cached data is available.
