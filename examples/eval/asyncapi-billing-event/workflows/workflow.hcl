# openapi = "asyncapi/events.yaml"
# http "publish_invoice_created"

  uws = "1.3.0"
  info {
    title       = "asyncapi_billing_event"
    description = "Publish a reviewed invoice-created event using a package-local AsyncAPI source."
    version     = "1.0.0"
  }
  sourceDescription "events" {
    url  = "asyncapi/events.yaml"
    type = "asyncapi"
  }
  operation "publish_invoice_created" {
    sourceDescription = "events"
    sourceOperationId = "publishInvoiceCreated"
    description       = "Publish the invoice-created event."
    request {
      body "customer_id" {
        __dollar__expr = "variables.inputs.customer_id"
      }
      body "invoice_id" {
        __dollar__expr = "variables.inputs.invoice_id"
      }
      header "trace_id" {
        __dollar__expr = "variables.inputs.trace_id"
      }
    }
  }
  workflow "main" {
    type        = "sequence"
    description = "Publish a reviewed invoice-created event using a package-local AsyncAPI source."
    outputs = {
      publish_result = "publish_invoice_created.received_body"
    }
    step "publish_invoice_created" {
      description  = "Publish the invoice-created event."
      operationRef = "publish_invoice_created"
    }
  }