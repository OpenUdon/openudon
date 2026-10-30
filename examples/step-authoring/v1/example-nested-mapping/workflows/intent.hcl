openapi = "openapi/project-api.yaml"

workflow {
  name        = "nested_project_inventory"
  description = "Check nested and renamed API field mappings."
}

input "pagination" {
  type     = "object"
  required = true
}

step "list_projects" {
  type      = "http"
  do        = "List projects visible to the configured account."
  operation = "listProjects"
  with = {
    "query.page_size" = "inputs.pagination.limit"
    X-API-Key         = "project_api_key"
  }
}

output "project_rows" {
  from = "list_projects.received_body.data.items"
}
