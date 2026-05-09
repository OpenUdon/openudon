# Timeout Idempotency Controls

Build a runtime-only workflow that submits one controlled local function call.

```openudon-policy
openapi: none required
runtimes:
  fnct: allowed
timeouts:
  workflow: 120
  steps:
    call_api: 10
idempotency:
  key: inputs.request_id
  onConflict: returnPrevious
  ttl: 86400
```

## Inputs

- request_id: string, required, idempotency identity for the logical workflow run
- payload: string, required, payload to submit

## Runtime Policy

- fnct is approved for this fixture.

## Data Flow

- Pass `inputs.payload` to `call_api.payload`.

## Outputs

- result: from `call_api.received_body`
