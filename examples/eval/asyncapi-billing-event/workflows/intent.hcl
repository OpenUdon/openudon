source = "asyncapi/events.yaml"
workflow {
  name        = "asyncapi_billing_event"
  description = "Publish a reviewed invoice-created event using a package-local AsyncAPI source."
}
input "invoice_id" {
  type     = "string"
  required = true
}
input "customer_id" {
  type     = "string"
  required = true
}
input "trace_id" {
  type     = "string"
  required = true
}
step "publish_invoice_created" {
  type = "http"
  do   = "Publish the invoice-created event."
  with = {
    "body.customer_id" = "inputs.customer_id"
    "body.invoice_id"  = "inputs.invoice_id"
    "header.trace_id"  = "inputs.trace_id"
  }
  source    = "asyncapi/events.yaml"
  operation = "publishInvoiceCreated"
}
output "publish_result" {
  from = "publish_invoice_created.received_body"
}
