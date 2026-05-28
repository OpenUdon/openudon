# gRPC Trace Export Eval

## Goal

Submit a reviewed OpenTelemetry trace export request using a package-local gRPC/protobuf source artifact

## Inputs

- `resource_spans`: required JSON-compatible trace resource spans payload for the export request.

## Outputs

- `export_result`: trace export result from the trusted runtime.

## External Systems and OpenAPI

- Use `grpc-protobuf/opentelemetry-trace-service-v1.proto` for gRPC/protobuf source metadata.
- OpenAPI: none required.
- OpenUdon validates and packages the gRPC/protobuf source-bound workflow, but RPC execution belongs to the trusted runtime.

## Runtime Policy

- `openapi` and `http` are allowed for source-bound API operations.
- `fnct`, `cmd`, and `ssh` are not required.

## Function Contracts

- No `fnct` helper contracts are required.

## Data Flow

- Pass `resource_spans` into the export request body field `resourceSpans`.

## Credentials and Secrets

- No credential values are stored in this provider-free fixture.
- Do not place collector credentials, API keys, OAuth tokens, TLS private keys, or other secret values in prompts, artifacts, or examples.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Real gRPC calls require human approval and trusted-runner execution.

## Fallback Behavior

- Stop if the gRPC method is unavailable or required trace payload details are missing.
