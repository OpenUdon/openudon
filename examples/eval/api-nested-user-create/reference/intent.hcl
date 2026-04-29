openapi = "openapi/users.yaml"

workflow {
  name        = "api_nested_user_create"
  description = "Create one sandbox user account with nested profile metadata."
}

input "email" {
  type     = "string"
  required = true
}

input "displayName" {
  type     = "string"
  required = true
}

input "department" {
  type     = "string"
  required = true
}

step "create_user" {
  type      = "http"
  do        = "Create one sandbox user."
  operation = "createUser"
  with = {
    email         = "inputs.email"
    displayName   = "inputs.displayName"
    profile       = "inputs.department"
    Authorization = "user_admin_token"
  }
}

output "user" {
  from = "create_user.received_body"
}
