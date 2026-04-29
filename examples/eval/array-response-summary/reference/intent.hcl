openapi = "openapi/alerts.yaml"

workflow {
  name        = "array_response_summary"
  description = "Fetch open alerts and render a local summary of affected services."
}

input "severity" {
  type     = "string"
  required = true
}

step "list_alerts" {
  type      = "http"
  do        = "List open alerts at or above the requested severity."
  operation = "listAlerts"
  with = {
    severity      = "inputs.severity"
    Authorization = "alerts_bearer_token"
  }
}

step "summarize_alerts" {
  type       = "fnct"
  do         = "Summarize alerts by affected service."
  depends_on = ["list_alerts"]
  bind {
    from = "list_alerts"
    fields = {
      alerts = "received_body.alerts"
    }
  }
}

output "summary" {
  from = "summarize_alerts.received_body"
}
