openapi = "openapi/airtable.yaml"

workflow {
  name        = "airtable_record_normalize"
  description = "Fetch one Airtable-style record and normalize response fields locally."
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
  do        = "Fetch one record from a sandbox base and table."
  operation = "getAirtableRecord"
  with = {
    baseId   = "inputs.baseId"
    tableId  = "inputs.tableId"
    recordId = "inputs.recordId"
  }
}

step "normalize_record" {
  type       = "fnct"
  do         = "Normalize the fetched record into a local payload."
  depends_on = ["get_record"]
  bind {
    from = "get_record"
    fields = {
      recordId    = "received_body.id"
      fields      = "received_body.fields"
      createdTime = "received_body.createdTime"
    }
  }
}

output "normalized_record" {
  from = "normalize_record.received_body"
}
