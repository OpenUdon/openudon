openapi = "openapi/crm.yaml"

workflow {
  name        = "crm_note_write"
  description = "Create an internal CRM note for a reviewed contact interaction."
}

input "contactId" {
  type     = "string"
  required = true
}

input "noteText" {
  type     = "string"
  required = true
}

step "create_contact_note" {
  type      = "http"
  do        = "Create an internal note on the CRM contact."
  operation = "createContactNote"
  with = {
    contactId  = "inputs.contactId"
    text       = "inputs.noteText"
    visibility = "internal"
  }
}

output "note" {
  from = "create_contact_note.received_body"
}
