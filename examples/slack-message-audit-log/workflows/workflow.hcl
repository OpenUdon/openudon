# openapi = "openapi/slack.yaml"
# http "post_message"
# fnct "render_audit_log"

  uws = "1.0.0"
  info {
    title       = "slack_message_audit_log"
    description = "Post a sandbox chat message and render a local audit log from the response."
    version     = "1.0.0"
  }
  sourceDescription "slack" {
    url  = "openapi/slack.yaml"
    type = "openapi"
  }
  operation "post_message" {
    sourceDescription  = "slack"
    openapiOperationId = "postMessage"
    description        = "Post one sandbox chat message."
    request {
      body "channel" {
        __dollar__expr = "variables.inputs.channel"
      }
      body "text" {
        __dollar__expr = "variables.inputs.text"
      }
    }
  }
  operation "render_audit_log" {
    description = "Render a local audit log from the post response."
    dependsOn   = ["post_message"]
    request {
      body "channel" {
        __dollar__expr = "post_message.received_body.channel"
      }
      body "ok" {
        __dollar__expr = "post_message.received_body.ok"
      }
      body "ts" {
        __dollar__expr = "post_message.received_body.ts"
      }
    }
    extensions {
      x-uws-operation-profile = "uws.runtime.1.0"
      x-uws-runtime {
        arguments = [
          {
            value = "post_message.received_body.channel"
            name = "channel"
          },
          {
            value = "post_message.received_body.ok"
            name = "ok"
          },
          {
            name = "ts"
            value = "post_message.received_body.ts"
          }
        ]
        function = "render_audit_log"
        type = "fnct"
      }
    }
  }
  workflow "main" {
    type        = "sequence"
    description = "Post a sandbox chat message and render a local audit log from the response."
    outputs = {
      audit_log = "render_audit_log.received_body"
    }
    step "post_message" {
      description  = "Post one sandbox chat message."
      operationRef = "post_message"
    }
    step "render_audit_log" {
      description  = "Render a local audit log from the post response."
      operationRef = "render_audit_log"
      dependsOn    = ["post_message"]
    }
  }