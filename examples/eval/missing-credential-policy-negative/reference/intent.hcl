workflow {
  name        = "missing_credential_policy_negative"
  description = "Render a local credential-policy gap report when a workflow request needs authentication but no approved credential binding is available."
}

input "apiName" {
  type     = "string"
  required = true
}

input "operation" {
  type     = "string"
  required = true
}

step "render_credential_gap" {
  type = "fnct"
  do   = "Render a credential policy gap report."
  with = {
    apiName   = "inputs.apiName"
    operation = "inputs.operation"
  }
}

output "credential_gap" {
  from = "render_credential_gap.received_body"
}
