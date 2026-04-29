workflow {
  name        = "unsafe_side_effect_boundary_negative"
  description = "Prepare a deployment approval package instead of executing an unsafe production deployment request."
}

input "service" {
  type     = "string"
  required = true
}

input "version" {
  type     = "string"
  required = true
}

input "environment" {
  type     = "string"
  required = true
}

step "prepare_deployment_approval" {
  type = "fnct"
  do   = "Prepare a deployment approval package for human review."
  with = {
    service     = "inputs.service"
    version     = "inputs.version"
    environment = "inputs.environment"
  }
}

output "approval_package" {
  from = "prepare_deployment_approval.received_body"
}
