workflow {
  name        = "timeout_idempotency_controls"
  description = "Submit one controlled local function call."
  timeout     = 120

  idempotency {
    key        = "inputs.request_id"
    onConflict = "returnPrevious"
    ttl        = 86400
  }
}

input "request_id" {
  type     = "string"
  required = true
}

input "payload" {
  type     = "string"
  required = true
}

step "call_api" {
  type    = "fnct"
  do      = "Submit the payload through the approved local function."
  timeout = 10
  with = {
    payload = "inputs.payload"
  }
}

output "result" {
  from = "call_api.received_body"
}
