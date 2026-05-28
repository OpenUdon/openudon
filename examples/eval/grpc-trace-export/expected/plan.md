# OpenUdon Workflow Plan

- Workflow: `grpc_trace_export`
- Summary: Submit a reviewed OpenTelemetry trace export request using a package-local gRPC/protobuf source artifact.
- Version: `openudon.workflow-plan.v1`

## Steps

- `export_trace` runtime `http` operation `opentelemetry.proto.collector.trace.v1.TraceService/Export`
  - binding: `resourceSpans <- inputs.resource_spans`
