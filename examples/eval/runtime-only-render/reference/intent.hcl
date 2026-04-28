workflow {
  name        = "runtime_only_render"
  description = "Render a local summary report."
}

input "summary" {
  type     = "string"
  required = true
}

step "render_report" {
  type = "fnct"
  do   = "Render the summary report."
  with = {
    summary = "inputs.summary"
  }
}

output "report" {
  from = "render_report.received_body"
}
