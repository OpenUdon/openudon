workflow {
  name        = "itops_slack_jira_issue_intake"
  description = "Convert one structured Slack issue report into a Jira issue and post a Slack confirmation."
}

input "channel" {
  type     = "string"
  required = true
}

input "messageTs" {
  type     = "string"
  required = true
}

input "projectKey" {
  type     = "string"
  required = true
}

step "get_slack_message" {
  type      = "http"
  do        = "Fetch the Slack issue report message."
  openapi   = "openapi/slack.yaml"
  operation = "getSlackMessage"
  with = {
    channel       = "inputs.channel"
    ts            = "inputs.messageTs"
    Authorization = "slack_bot_token"
  }
}

step "parse_issue_report" {
  type       = "fnct"
  do         = "Parse title, description, priority, and issue type from Slack text."
  depends_on = ["get_slack_message"]
  bind {
    from = "get_slack_message"
    fields = {
      text = "received_body.message.text"
    }
  }
}

step "create_jira_issue" {
  type       = "http"
  do         = "Create the Jira issue from parsed Slack report fields."
  openapi    = "openapi/jira.yaml"
  operation  = "createIssue"
  depends_on = ["parse_issue_report"]
  with = {
    projectKey    = "inputs.projectKey"
    Authorization = "jira_api_token"
  }
  bind {
    from = "parse_issue_report"
    fields = {
      summary     = "received_body.title"
      description = "received_body.description"
      priority    = "received_body.priority"
      issueType   = "received_body.issueType"
    }
  }
}

step "post_slack_confirmation" {
  type       = "http"
  do         = "Post a Slack confirmation with the Jira issue key."
  openapi    = "openapi/slack.yaml"
  operation  = "postMessage"
  depends_on = ["create_jira_issue"]
  with = {
    channel       = "inputs.channel"
    Authorization = "slack_bot_token"
  }
  bind {
    from = "create_jira_issue"
    fields = {
      text = "received_body.key"
    }
  }
}

output "issue_key" {
  from = "create_jira_issue.received_body.key"
}
