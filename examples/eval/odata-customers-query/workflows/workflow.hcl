# source = "odata/odata-operations-service.xml"
# http "query_customers"

  uws = "1.4.0"
  info {
    title       = "odata_customers_query"
    description = "Read a reviewed Customers entity set query using a package-local OData CSDL artifact."
    version     = "1.0.0"
  }
  sourceDescription "odata_operations_service" {
    url  = "odata/odata-operations-service.xml"
    type = "odata"
  }
  operation "query_customers" {
    sourceDescription = "odata_operations_service"
    sourceOperationId = "entitySet.Customers.query"
    description       = "Read the Customers entity set."
    request {
      query "__dollar__top" {
        __dollar__expr = "variables.inputs.max_customers"
      }
    }
  }
  workflow "main" {
    type        = "sequence"
    description = "Read a reviewed Customers entity set query using a package-local OData CSDL artifact."
    outputs = {
      customers_result = "query_customers.received_body"
    }
    step "query_customers" {
      description  = "Read the Customers entity set."
      operationRef = "query_customers"
    }
  }