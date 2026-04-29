workflow {
  name        = "webhook_validation_fnct"
  description = "Validate an incoming webhook payload and render an accepted payload summary without external API calls."
}

input "payload" {
  type     = "object"
  required = true
}

input "signature" {
  type     = "string"
  required = true
}

step "validate_webhook" {
  type = "fnct"
  do   = "Validate the webhook signature and normalize the payload."
  with = {
    payload   = "inputs.payload"
    signature = "inputs.signature"
  }
}

output "validation" {
  from = "validate_webhook.received_body"
}
