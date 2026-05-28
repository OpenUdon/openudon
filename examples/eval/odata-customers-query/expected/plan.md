# OpenUdon Workflow Plan

- Workflow: `odata_customers_query`
- Summary: Read a reviewed Customers entity set query using a package-local OData CSDL artifact.
- Version: `openudon.workflow-plan.v1`

## Steps

- `query_customers` runtime `http` operation `entitySet.Customers.query`
  - binding: `$top <- inputs.max_customers`
