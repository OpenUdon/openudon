openapi = "openapi/gmail.json"

workflow {
  name        = "n8n_gmail_message_send"
  description = "Represent the n8n Gmail message send workflow as a OpenUdon intent workflow."
}

input "sendTo" {
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
  do        = "Send one Gmail message."
  operation = "sendMessage"
  with = {
    sendTo  = "inputs.sendTo"
    subject = "inputs.subject"
    message = "inputs.message"
  }
}

output "sent_message" {
  from = "send_message.received_body"
}
