workflow {
  name        = "cmd_disallowed_deploy"
  description = "This negative fixture should fail if a cmd step is generated."
}

step "check_deploy_status" {
  type = "cmd"
  do   = "Run the deployment status command."
}

output "status" {
  from = "check_deploy_status.received_body"
}
