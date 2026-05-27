# OpenUdon Workflow Plan

- Workflow: `asyncapi_billing_event`
- Summary: Publish a reviewed invoice-created event using a package-local AsyncAPI source.
- Version: `openudon.workflow-plan.v1`

## Steps

- `publish_invoice_created` runtime `http` operation `publishInvoiceCreated`
  - binding: `body.customer_id <- inputs.customer_id`
  - binding: `body.invoice_id <- inputs.invoice_id`
  - binding: `header.trace_id <- inputs.trace_id`
