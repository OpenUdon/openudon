workflow {
  name        = "cmd_allowed_deploy"
  description = "Run the approved deployment status command."
}

step "check_deploy_status" {
  type = "cmd"
  do   = "Run the sandbox deployment status command."
}

output "status" {
  from = "check_deploy_status.received_body"
}
