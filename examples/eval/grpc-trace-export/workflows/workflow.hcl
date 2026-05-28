# source = "grpc-protobuf/opentelemetry-trace-service-v1.proto"
# http "export_trace"

  uws = "1.4.0"
  info {
    title       = "grpc_trace_export"
    description = "Submit a reviewed OpenTelemetry trace export request using a package-local gRPC/protobuf source artifact."
    version     = "1.0.0"
  }
  sourceDescription "opentelemetry_trace_service_v1" {
    url  = "grpc-protobuf/opentelemetry-trace-service-v1.proto"
    type = "grpc-protobuf"
  }
  operation "export_trace" {
    sourceDescription = "opentelemetry_trace_service_v1"
    sourceOperationId = "opentelemetry.proto.collector.trace.v1.TraceService/Export"
    description       = "Submit the OpenTelemetry trace export request."
    request {
      body "resourceSpans" {
        __dollar__expr = "variables.inputs.resource_spans"
      }
    }
  }
  workflow "main" {
    type        = "sequence"
    description = "Submit a reviewed OpenTelemetry trace export request using a package-local gRPC/protobuf source artifact."
    outputs = {
      export_result = "export_trace.received_body"
    }
    step "export_trace" {
      description  = "Submit the OpenTelemetry trace export request."
      operationRef = "export_trace"
    }
  }