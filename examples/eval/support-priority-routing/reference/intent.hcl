openapi = "openapi/support.yaml"

workflow {
  name        = "support_priority_routing"
  description = "Fetch a support ticket, classify priority, and prepare one internal routing result."
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

step "classify_priority" {
  type       = "fnct"
  do         = "Classify the support ticket priority."
  depends_on = ["get_ticket"]
  bind {
    from = "get_ticket"
    fields = {
      ticket = "received_body"
    }
  }
}

step "prepare_routing_result" {
  type       = "fnct"
  do         = "Prepare the selected internal handling result from ticket priority."
  depends_on = ["get_ticket", "classify_priority"]
  bind {
    from = "get_ticket"
    fields = {
      ticket = "received_body"
    }
  }
  bind {
    from = "classify_priority"
    fields = {
      classification = "received_body"
    }
  }
}

output "routing_result" {
  from = "prepare_routing_result.received_body"
}
