# OpenRPC Simple Math Addition Eval

## Goal

Run the reviewed simple-math addition method using a package-local OpenRPC artifact

## Inputs

- `left_number`: required number for JSON-RPC parameter `a`.
- `right_number`: required number for JSON-RPC parameter `b`.

## Outputs

- `addition_result`: addition result from the trusted runtime.

## External Systems and OpenAPI

- Use `openrpc/openrpc-simple-math.json` for JSON-RPC source metadata.
- OpenAPI: none required.
- OpenUdon validates and packages the OpenRPC source-bound workflow, but JSON-RPC execution belongs to the trusted runtime.

## Runtime Policy

- `openapi` and `http` are allowed for source-bound API operations.
- `fnct`, `cmd`, and `ssh` are not required.

## Function Contracts

- No `fnct` helper contracts are required.

## Data Flow

- Pass `left_number` to JSON-RPC parameter `a`.
- Pass `right_number` to JSON-RPC parameter `b`.

## Credentials and Secrets

- No credential values are stored in this provider-free fixture.
- Do not place JSON-RPC endpoint credentials, API keys, OAuth tokens, or other secret values in prompts, artifacts, or examples.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Real JSON-RPC calls require human approval and trusted-runner execution.

## Fallback Behavior

- Stop if the OpenRPC method is unavailable or required operands are missing.
