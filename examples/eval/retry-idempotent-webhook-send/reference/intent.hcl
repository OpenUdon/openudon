openapi = "openapi/webhooks.yaml"

workflow {
  name        = "retry_idempotent_webhook_send"
  description = "Send one idempotent webhook notification with explicit timeout and idempotency metadata."
  timeout     = 120
  idempotency = {
    key         = "inputs.eventId"
    onConflict  = "returnPrevious"
    ttl         = 86400
  }
}

input "eventId" {
  type     = "string"
  required = true
}

input "payload" {
  type     = "object"
  required = true
}

step "send_webhook" {
  type      = "http"
  do        = "Send the idempotent webhook payload."
  operation = "sendWebhook"
  timeout   = 20
  with = {
    idempotencyKey = "inputs.eventId"
    payload        = "inputs.payload"
    Authorization  = "webhook_bearer_token"
  }
}

output "delivery" {
  from = "send_webhook.received_body"
}
