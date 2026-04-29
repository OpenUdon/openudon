openapi = "openapi/compliance.yaml"

workflow {
  name        = "api_header_key_report"
  description = "Fetch a compliance report using an API key carried in a request header."
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
    reportId     = "inputs.reportId"
    api_key_auth = "compliance_api_key"
  }
}

output "report" {
  from = "get_report.received_body"
}
