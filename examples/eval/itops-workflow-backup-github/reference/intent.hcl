workflow {
  name        = "itops_workflow_backup_github"
  description = "Back up one n8n workflow definition into a GitHub repository path."
}

input "workflowId" {
  type     = "string"
  required = true
}

input "repoOwner" {
  type     = "string"
  required = true
}

input "repoName" {
  type     = "string"
  required = true
}

input "repoPath" {
  type     = "string"
  required = true
}

step "get_workflow" {
  type      = "http"
  do        = "Export one workflow from n8n."
  openapi   = "openapi/n8n.yaml"
  operation = "getWorkflow"
  with = {
    workflowId   = "inputs.workflowId"
    n8n_api_key  = "n8n_api_key"
  }
}

step "render_backup_file" {
  type       = "fnct"
  do         = "Render the exported workflow into a deterministic repository file."
  depends_on = ["get_workflow"]
  with = {
    repoPath = "inputs.repoPath"
  }
  bind {
    from = "get_workflow"
    fields = {
      workflow = "received_body"
    }
  }
}

step "get_existing_backup" {
  type       = "http"
  do         = "Fetch existing GitHub file metadata if present."
  openapi    = "openapi/github.yaml"
  operation  = "getContent"
  depends_on = ["render_backup_file"]
  with = {
    owner         = "inputs.repoOwner"
    repo          = "inputs.repoName"
    Authorization = "github_token"
  }
  bind {
    from = "render_backup_file"
    fields = {
      path = "received_body.path"
    }
  }
}

step "upsert_backup_file" {
  type       = "http"
  do         = "Create or update the GitHub backup file."
  openapi    = "openapi/github.yaml"
  operation  = "putContent"
  depends_on = ["get_existing_backup", "render_backup_file"]
  with = {
    owner         = "inputs.repoOwner"
    repo          = "inputs.repoName"
    message       = "Back up n8n workflow"
    Authorization = "github_token"
  }
  bind {
    from = "render_backup_file"
    fields = {
      path    = "received_body.path"
      content = "received_body.content"
    }
  }
  bind {
    from = "get_existing_backup"
    fields = {
      sha = "received_body.sha"
    }
  }
}

output "backup_commit" {
  from = "upsert_backup_file.received_body.commit.sha"
}
