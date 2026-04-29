openapi = "openapi/google_drive.json"

workflow {
  name        = "n8n_google_drive_file_upload"
  description = "Represent the n8n Google Drive file upload workflow as a Ramen intent workflow."
}

input "name" {
  type     = "string"
  required = true
}

input "data" {
  type     = "string"
  required = true
}

step "upload_file" {
  type      = "http"
  do        = "Upload one file to Google Drive."
  operation = "uploadFile"
  with = {
    name = "inputs.name"
    data = "inputs.data"
  }
}

output "file" {
  from = "upload_file.received_body"
}
