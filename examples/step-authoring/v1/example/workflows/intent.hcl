openapi = "openapi/project-api.yaml"

workflow {
  name        = "step_check_example"
  description = "A local example for deterministic step-check conformance."
}

input "page_size" {
  type     = "integer"
  required = true
}

step "list_projects" {
  type      = "http"
  do        = "List projects visible to the configured account."
  operation = "listProjects"
  with = {
    page_size = "inputs.page_size"
    X-API-Key = "project_api_key"
  }
}

output "projects" {
  from = "list_projects.received_body.projects"
}
