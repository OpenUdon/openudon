openapi = "openapi/compliance.yaml"

workflow {
  name        = "compliance_report_summary"
  description = "Fetch a compliance report and render a local status summary from the API response."
}

input "reportId" {
  type     = "string"
  required = true
}

step "get_report" {
  type      = "http"
  do        = "Fetch the compliance report."
  operation = "getComplianceReport"
  with = {
    reportId = "inputs.reportId"
  }
}

step "render_report_summary" {
  type       = "fnct"
  do         = "Render the compliance report status summary."
  depends_on = ["get_report"]
  bind {
    from = "get_report"
    fields = {
      reportId   = "received_body.id"
      status     = "received_body.status"
      ownerEmail = "received_body.ownerEmail"
    }
  }
}

output "summary" {
  from = "render_report_summary.received_body"
}
