openapi = "openapi/audit-events.yaml"

workflow {
  name        = "cursor_pagination_report"
  description = "Fetch two cursor-paginated audit event pages and render a local report."
}

step "list_events_first_page" {
  type      = "http"
  do        = "List the first page of audit events."
  operation = "listAuditEvents"
  with = {
    limit         = "100"
    Authorization = "audit_events_bearer_token"
  }
}

step "list_events_second_page" {
  type       = "http"
  do         = "List the second page of audit events using the cursor from the first page."
  operation  = "listAuditEvents"
  depends_on = ["list_events_first_page"]
  with = {
    limit         = "100"
    Authorization = "audit_events_bearer_token"
  }
  bind {
    from = "list_events_first_page"
    fields = {
      cursor = "received_body.page.nextCursor"
    }
  }
}

step "render_audit_report" {
  type       = "fnct"
  do         = "Render the two fetched audit event pages into one report."
  depends_on = ["list_events_first_page", "list_events_second_page"]
  bind {
    from = "list_events_first_page"
    fields = {
      page_1_events = "received_body.events"
    }
  }
  bind {
    from = "list_events_second_page"
    fields = {
      page_2_events = "received_body.events"
    }
  }
}

output "report" {
  from = "render_audit_report.received_body"
}
