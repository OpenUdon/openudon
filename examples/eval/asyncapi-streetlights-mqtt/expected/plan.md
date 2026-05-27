# OpenUdon Workflow Plan

- Workflow: `asyncapi_streetlights_mqtt`
- Summary: Publish a reviewed Streetlights MQTT dim-light command using a package-local AsyncAPI source.
- Version: `openudon.workflow-plan.v1`

## Steps

- `dim_streetlight` runtime `http` operation `dimLight`
  - binding: `body.percentage <- inputs.dim_percentage`
  - binding: `body.sentAt <- inputs.sent_at`
  - binding: `header.my-app-header <- inputs.app_header_value`
  - binding: `path.streetlightId <- inputs.streetlight_id`
