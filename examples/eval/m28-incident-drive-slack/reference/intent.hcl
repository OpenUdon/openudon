workflow {
  name        = "m28_incident_drive_slack_archive"
  description = "M28 sample: create an incident ticket, alert Slack, and archive an incident timeline report to Drive."
}

input "service" {
  type     = "string"
  required = true
}

input "severity" {
  type     = "string"
  required = true
}

input "description" {
  type     = "string"
  required = true
}

input "timeline" {
  type     = "string"
  required = true
}

input "slackChannel" {
  type     = "string"
  required = true
}

input "driveFolderId" {
  type     = "string"
  required = true
}

step "create_jira_incident" {
  type      = "http"
  do        = "Create the Jira incident issue."
  openapi   = "openapi/jira.yaml"
  operation = "createIssue"
  with = {
    service       = "inputs.service"
    severity      = "inputs.severity"
    description   = "inputs.description"
    summary       = "inputs.service"
    Authorization = "jira_api_token"
  }
}

step "format_slack_alert" {
  type       = "fnct"
  do         = "Format the on-call Slack alert."
  depends_on = ["create_jira_incident"]
  with = {
    service     = "inputs.service"
    severity    = "inputs.severity"
    description = "inputs.description"
  }
  bind {
    from = "create_jira_incident"
    fields = {
      issueKey = "received_body.key"
      issueURL = "received_body.self"
    }
  }
}

step "post_slack_alert" {
  type       = "http"
  do         = "Post the incident alert to Slack."
  openapi    = "openapi/slack.yaml"
  operation  = "postMessage"
  depends_on = ["format_slack_alert"]
  with = {
    channel       = "inputs.slackChannel"
    Authorization = "slack_bot_token"
  }
  bind {
    from = "format_slack_alert"
    fields = {
      text = "received_body.text"
    }
  }
}

step "render_timeline_report" {
  type       = "fnct"
  do         = "Render the incident timeline report for archival."
  depends_on = ["create_jira_incident", "post_slack_alert"]
  with = {
    service     = "inputs.service"
    severity    = "inputs.severity"
    description = "inputs.description"
    timeline    = "inputs.timeline"
  }
  bind {
    from = "create_jira_incident"
    fields = {
      issueKey = "received_body.key"
    }
  }
}

step "upload_timeline_report" {
  type       = "http"
  do         = "Upload the incident timeline report to Google Drive."
  openapi    = "openapi/drive.yaml"
  operation  = "uploadFile"
  depends_on = ["render_timeline_report"]
  with = {
    parentId      = "inputs.driveFolderId"
    Authorization = "google_drive_oauth_token"
  }
  bind {
    from = "render_timeline_report"
    fields = {
      name    = "received_body.name"
      content = "received_body.content"
    }
  }
}

output "archive_file" {
  from = "upload_timeline_report.received_body"
}
