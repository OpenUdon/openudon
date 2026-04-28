workflow {
  name        = "profile_boundary_manifest"
  description = "Prepare a local export manifest without invoking direct profile execution."
}

input "dataset" {
  type     = "string"
  required = true
}

input "since" {
  type     = "string"
  required = true
}

step "render_export_manifest" {
  type = "fnct"
  do   = "Render a local database export request manifest for review."
  with = {
    dataset = "inputs.dataset"
    since   = "inputs.since"
  }
}

output "manifest" {
  from = "render_export_manifest.received_body"
}
