openapi = "openapi/slack.yaml"
workflow {
  name        = "slack_message_audit_log"
  description = "Post a sandbox chat message and render a local audit log from the response."
}
input "channel" {
  type     = "string"
  required = true
}
input "text" {
  type     = "string"
  required = true
}
step "post_message" {
  type = "http"
  do   = "Post one sandbox chat message."
  with = {
    channel = "inputs.channel"
    text    = "inputs.text"
  }
  operation = "postMessage"
}
step "render_audit_log" {
  type       = "fnct"
  do         = "Render a local audit log from the post response."
  depends_on = ["post_message"]
  bind {
    from = "post_message"
    fields = {
      channel = "received_body.channel"
      ok      = "received_body.ok"
      ts      = "received_body.ts"
    }
  }
}
output "audit_log" {
  from = "render_audit_log.received_body"
}
