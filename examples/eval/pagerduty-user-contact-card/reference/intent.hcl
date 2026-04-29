openapi = "openapi/pagerduty.yaml"

workflow {
  name        = "pagerduty_user_contact_card"
  description = "Fetch one PagerDuty-style user and render a local contact card."
}

input "userId" {
  type     = "string"
  required = true
}

step "get_user" {
  type      = "http"
  do        = "Fetch one user profile."
  operation = "getUser"
  with = {
    userId = "inputs.userId"
  }
}

step "render_contact_card" {
  type       = "fnct"
  do         = "Render a contact card from the user response."
  depends_on = ["get_user"]
  bind {
    from = "get_user"
    fields = {
      userId = "received_body.user.id"
      name   = "received_body.user.name"
      email  = "received_body.user.email"
    }
  }
}

output "contact_card" {
  from = "render_contact_card.received_body"
}
