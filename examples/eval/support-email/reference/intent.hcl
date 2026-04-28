openapi = "openapi/support.yaml"

workflow {
  name        = "support_email"
  description = "Fetch support ticket details and send a confirmation email."
}

input "ticketId" {
  type     = "string"
  required = true
}

step "get_ticket" {
  type      = "http"
  do        = "Fetch support ticket details."
  operation = "getTicket"
  with = {
    ticketId = "inputs.ticketId"
  }
}

step "send_confirmation_email" {
  type       = "fnct"
  do         = "Send the confirmation email through the approved adapter."
  depends_on = ["get_ticket"]
  bind {
    from = "get_ticket"
    fields = {
      to      = "received_body.requesterEmail"
      subject = "received_body.subject"
      body    = "received_body.summary"
    }
  }
}

output "email_status" {
  from = "send_confirmation_email.received_body"
}
