openapi = "openapi/jira.json"

workflow {
  name        = "n8n_jira_issue_get"
  description = "Represent the scanner-backed Jira issue get slice as a OpenUdon intent workflow."
}

input "issueKey" {
  type     = "string"
  required = true
}

step "get_issue" {
  type      = "http"
  do        = "Fetch one Jira issue."
  operation = "getIssue"
  with = {
    issueKey = "inputs.issueKey"
  }
}

output "issue" {
  from = "get_issue.received_body"
}
