openapi = "openapi/audit.yaml"

workflow {
  name        = "offset_pagination_export"
  description = "Fetch two offset-paginated pages of audit records and merge them into one export payload."
}

step "list_records_page_1" {
  type      = "http"
  do        = "Fetch the first offset page."
  operation = "listAuditRecords"
  with = {
    offset        = "0"
    limit         = "100"
    Authorization = "audit_bearer_token"
  }
}

step "list_records_page_2" {
  type      = "http"
  do        = "Fetch the second offset page."
  operation = "listAuditRecords"
  with = {
    offset        = "100"
    limit         = "100"
    Authorization = "audit_bearer_token"
  }
}

step "merge_audit_pages" {
  type       = "fnct"
  do         = "Merge audit records from both pages."
  depends_on = ["list_records_page_1", "list_records_page_2"]
  bind {
    from = "list_records_page_1"
    fields = {
      page_1 = "received_body.records"
    }
  }
  bind {
    from = "list_records_page_2"
    fields = {
      page_2 = "received_body.records"
    }
  }
}

output "export_payload" {
  from = "merge_audit_pages.received_body"
}
