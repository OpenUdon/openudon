openapi = "openapi/slack.json"

workflow {
  name        = "n8n_slack_message_post"
  description = "Represent the n8n Slack message post workflow as a Ramen intent workflow."
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
  type      = "http"
  do        = "Post a Slack message."
  operation = "postMessage"
  with = {
    channel = "inputs.channel"
    text    = "inputs.text"
  }
}

output "message" {
  from = "post_message.received_body"
}
