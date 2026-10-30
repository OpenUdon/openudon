workflow {
  name        = "project_inventory"
  description = "Read a project inventory from an approved source"
}
input "page_size" {
  type     = "integer"
  required = true
}
step "list_projects" {
  type = "http"
  do   = "List projects visible to the configured account"
  with = {
    X-API-Key = "credentials.project_api_key"
    page_size = "inputs.page_size"
  }
  source    = "openapi/project-api.yaml"
  operation = "listProjects"
}
output "projects" {
  from = "list_projects.received_body.projects"
}
