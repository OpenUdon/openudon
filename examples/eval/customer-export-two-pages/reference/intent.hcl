openapi = "openapi/customers.yaml"

workflow {
  name        = "customer_export_two_pages"
  description = "Fetch two pages of customers and merge them into one export payload."
}

step "list_customers_page_1" {
  type      = "http"
  do        = "List the first page of customers."
  operation = "listCustomers"
  with = {
    page  = "1"
    limit = "50"
  }
}

step "list_customers_page_2" {
  type      = "http"
  do        = "List the second page of customers."
  operation = "listCustomers"
  with = {
    page  = "2"
    limit = "50"
  }
}

step "merge_customer_pages" {
  type       = "fnct"
  do         = "Merge customer records from the first two pages."
  depends_on = ["list_customers_page_1", "list_customers_page_2"]
  bind {
    from = "list_customers_page_1"
    fields = {
      page_1 = "received_body.customers"
    }
  }
  bind {
    from = "list_customers_page_2"
    fields = {
      page_2 = "received_body.customers"
    }
  }
}

output "export_payload" {
  from = "merge_customer_pages.received_body"
}
