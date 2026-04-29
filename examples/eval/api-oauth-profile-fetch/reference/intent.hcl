openapi = "openapi/directory.yaml"

workflow {
  name        = "api_oauth_profile_fetch"
  description = "Fetch one employee profile using a bearer credential binding."
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
    employeeId    = "inputs.employeeId"
    Authorization = "directory_bearer_token"
  }
}

output "profile" {
  from = "get_profile.received_body"
}
