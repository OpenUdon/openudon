workflow {
  name        = "incomplete_brief_repair"
  description = "Render a local clarification checklist for an incomplete workflow brief."
}

input "brief" {
  type     = "string"
  required = true
}

step "render_clarification_checklist" {
  type = "fnct"
  do   = "Render missing workflow requirements as a clarification checklist."
  with = {
    brief = "inputs.brief"
  }
}

output "clarification_checklist" {
  from = "render_clarification_checklist.received_body"
}
