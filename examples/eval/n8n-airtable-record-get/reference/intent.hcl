openapi = "openapi/airtable.json"

workflow {
  name        = "n8n_airtable_record_get"
  description = "Represent the n8n Airtable record get workflow as a OpenUdon intent workflow."
}

input "baseId" {
  type     = "string"
  required = true
}

input "tableId" {
  type     = "string"
  required = true
}

input "recordId" {
  type     = "string"
  required = true
}

step "get_record" {
  type      = "http"
  do        = "Fetch one Airtable record."
  operation = "getAirtableRecord"
  with = {
    baseId  = "inputs.baseId"
    tableId = "inputs.tableId"
    id      = "inputs.recordId"
  }
}

output "record" {
  from = "get_record.received_body"
}
