openapi = "openapi/directory.yaml"

workflow {
  name        = "profile_fetch_access_card"
  description = "Fetch an employee profile and render an access review card from the returned profile fields."
}

input "employeeId" {
  type     = "string"
  required = true
}

step "get_profile" {
  type      = "http"
  do        = "Fetch the employee profile."
  operation = "getEmployeeProfile"
  with = {
    employeeId = "inputs.employeeId"
  }
}

step "render_access_card" {
  type       = "fnct"
  do         = "Render an access review card from profile fields."
  depends_on = ["get_profile"]
  bind {
    from = "get_profile"
    fields = {
      email        = "received_body.email"
      managerEmail = "received_body.managerEmail"
      department   = "received_body.department"
    }
  }
}

output "access_card" {
  from = "render_access_card.received_body"
}
