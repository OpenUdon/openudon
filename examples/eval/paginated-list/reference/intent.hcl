openapi = "openapi/customers.yaml"

workflow {
  name        = "paginated_list"
  description = "List the first page of customers."
}

step "list_customers" {
  type      = "http"
  do        = "List customer records."
  operation = "listCustomers"
  with = {
    page  = "1"
    limit = "50"
  }
}

output "customers" {
  from = "list_customers.received_body"
}
