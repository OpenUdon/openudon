openapi = "openapi/pagerduty.json"

workflow {
  name        = "n8n_pagerduty_user_get"
  description = "Represent the n8n PagerDuty user get workflow as a Ramen intent workflow."
}

input "userId" {
  type     = "string"
  required = true
}

step "get_user" {
  type      = "http"
  do        = "Fetch one PagerDuty user."
  operation = "getUser"
  with = {
    userId = "inputs.userId"
  }
}

output "user" {
  from = "get_user.received_body"
}
