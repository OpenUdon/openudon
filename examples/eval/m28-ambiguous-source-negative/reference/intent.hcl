workflow {
  name        = "m28_ambiguous_source_negative"
  description = "M28 negative sample: stop and render a local gap report when an ambiguous provider action lacks usable API source evidence."
}

input "provider" {
  type     = "string"
  required = true
}

input "action" {
  type     = "string"
  required = true
}

step "render_capability_gap" {
  type = "fnct"
  do   = "Render a capability gap report for the missing provider action."
  with = {
    provider = "inputs.provider"
    action   = "inputs.action"
  }
}

output "gap_report" {
  from = "render_capability_gap.received_body"
}
