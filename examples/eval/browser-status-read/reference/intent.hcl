source = "browser-profiles/status.json"

workflow {
  name        = "browser_status_read"
  description = "Read the reviewed status banner from a UI-only provider capability."
}

input "item" {
  type     = "string"
  required = true
}

step "read" {
  type      = "browser"
  source    = "browser-profiles/status.json"
  operation = "read_status"
  with = {
    item = "inputs.item"
  }
}

output "status" {
  from = "read.received_body.status"
}
