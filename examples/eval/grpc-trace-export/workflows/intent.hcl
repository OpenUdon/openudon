source = "grpc-protobuf/opentelemetry-trace-service-v1.proto"
workflow {
  name        = "grpc_trace_export"
  description = "Submit a reviewed OpenTelemetry trace export request using a package-local gRPC/protobuf source artifact."
}
input "resource_spans" {
  type     = "object"
  required = true
}
step "export_trace" {
  type = "http"
  do   = "Submit the OpenTelemetry trace export request."
  with = {
    resourceSpans = "inputs.resource_spans"
  }
  source    = "grpc-protobuf/opentelemetry-trace-service-v1.proto"
  operation = "opentelemetry.proto.collector.trace.v1.TraceService/Export"
}
output "export_result" {
  from = "export_trace.received_body"
}
