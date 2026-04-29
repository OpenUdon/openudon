openapi = "openapi/gmail.yaml"

workflow {
  name        = "gmail_send_audit_receipt"
  description = "Send a sandbox email and render a local audit receipt from the send response."
}

input "to" {
  type     = "string"
  required = true
}

input "subject" {
  type     = "string"
  required = true
}

input "message" {
  type     = "string"
  required = true
}

step "send_message" {
  type      = "http"
  do        = "Send one sandbox email message."
  operation = "sendMessage"
  with = {
    to      = "inputs.to"
    subject = "inputs.subject"
    message = "inputs.message"
  }
}

step "render_audit_receipt" {
  type       = "fnct"
  do         = "Render a local audit receipt from the sent message response."
  depends_on = ["send_message"]
  bind {
    from = "send_message"
    fields = {
      messageId = "received_body.id"
      threadId  = "received_body.threadId"
    }
  }
}

output "audit_receipt" {
  from = "render_audit_receipt.received_body"
}
