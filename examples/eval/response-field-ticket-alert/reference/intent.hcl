openapi = "openapi/tickets.yaml"

workflow {
  name        = "response_field_ticket_alert"
  description = "Fetch a ticket, extract nested assignee data, and send an internal alert."
}

input "ticketId" {
  type     = "string"
  required = true
}

step "get_ticket" {
  type      = "http"
  do        = "Fetch ticket details."
  operation = "getTicket"
  with = {
    ticketId      = "inputs.ticketId"
    Authorization = "tickets_bearer_token"
  }
}

step "send_assignee_alert" {
  type       = "fnct"
  do         = "Send an internal alert to the ticket assignee."
  depends_on = ["get_ticket"]
  bind {
    from = "get_ticket"
    fields = {
      to      = "received_body.assignee.email"
      subject = "received_body.subject"
      body    = "received_body.summary"
    }
  }
}

output "send_status" {
  from = "send_assignee_alert.received_body"
}
