source = "odata/odata-operations-service.xml"
workflow {
  name        = "odata_customers_query"
  description = "Read a reviewed Customers entity set query using a package-local OData CSDL artifact."
}
input "max_customers" {
  type     = "number"
  required = true
}
step "query_customers" {
  type = "http"
  do   = "Read the Customers entity set."
  with = {
    "$top" = "inputs.max_customers"
  }
  source    = "odata/odata-operations-service.xml"
  operation = "entityset.Customers"
}
output "customers_result" {
  from = "query_customers.received_body"
}
