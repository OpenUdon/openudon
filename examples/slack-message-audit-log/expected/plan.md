# OpenUdon Workflow Plan

- Workflow: `slack_message_audit_log`
- Summary: Post a sandbox chat message and render a local audit log from the response.
- Version: `openudon.workflow-plan.v1`

## Steps

- `post_message` runtime `http` operation `postMessage`
  - binding: `channel <- inputs.channel`
  - binding: `text <- inputs.text`
- `render_audit_log` runtime `fnct`
  - depends_on: `post_message`
  - binding: `channel <- received_body.channel`
  - binding: `ok <- received_body.ok`
  - binding: `ts <- received_body.ts`
