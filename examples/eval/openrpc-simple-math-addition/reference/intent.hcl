source = "openrpc/openrpc-simple-math.json"
workflow {
  name        = "openrpc_simple_math_addition"
  description = "Run the reviewed simple-math addition method using a package-local OpenRPC artifact."
}
input "left_number" {
  type     = "number"
  required = true
}
input "right_number" {
  type     = "number"
  required = true
}
step "add_numbers" {
  type = "http"
  do   = "Run the simple-math addition method."
  with = {
    "a" = "inputs.left_number"
    "b" = "inputs.right_number"
  }
  source    = "openrpc/openrpc-simple-math.json"
  operation = "addition"
}
output "addition_result" {
  from = "add_numbers.received_body"
}
