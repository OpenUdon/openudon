openapi = "openapi/users.yaml"

workflow {
  name        = "user_create_welcome_message"
  description = "Create a sandbox user account and render a welcome message from the created user response."
}

input "email" {
  type     = "string"
  required = true
}

input "displayName" {
  type     = "string"
  required = true
}

step "create_user" {
  type      = "http"
  do        = "Create one sandbox user."
  operation = "createUser"
  with = {
    email       = "inputs.email"
    displayName = "inputs.displayName"
  }
}

step "render_welcome_message" {
  type       = "fnct"
  do         = "Render a welcome message from the created user response."
  depends_on = ["create_user"]
  bind {
    from = "create_user"
    fields = {
      userId      = "received_body.id"
      email       = "received_body.email"
      displayName = "received_body.displayName"
    }
  }
}

output "welcome_message" {
  from = "render_welcome_message.received_body"
}
