workflow {
  name        = "missing_openapi_capability_negative"
  description = "Record that a requested provider action has no available OpenAPI evidence and render a local gap report instead of inventing an API call."
}

input "provider" {
  type     = "string"
  required = true
}

input "action" {
  type     = "string"
  required = true
}

step "render_capability_gap" {
  type = "fnct"
  do   = "Render a capability gap report for the missing provider action."
  with = {
    provider = "inputs.provider"
    action   = "inputs.action"
  }
}

output "gap_report" {
  from = "render_capability_gap.received_body"
}
