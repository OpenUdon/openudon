# source = "openrpc/openrpc-simple-math.json"
# http "add_numbers"

  uws = "1.4.0"
  info {
    title       = "openrpc_simple_math_addition"
    description = "Run the reviewed simple-math addition method using a package-local OpenRPC artifact."
    version     = "1.0.0"
  }
  sourceDescription "openrpc_simple_math" {
    url  = "openrpc/openrpc-simple-math.json"
    type = "openrpc"
  }
  operation "add_numbers" {
    sourceDescription = "openrpc_simple_math"
    sourceOperationId = "addition"
    description       = "Run the simple-math addition method."
    request {
      body "params" "a" {
        __dollar__expr = "variables.inputs.left_number"
      }
      body "params" "b" {
        __dollar__expr = "variables.inputs.right_number"
      }
    }
  }
  workflow "main" {
    type        = "sequence"
    description = "Run the reviewed simple-math addition method using a package-local OpenRPC artifact."
    outputs = {
      addition_result = "add_numbers.received_body"
    }
    step "add_numbers" {
      description  = "Run the simple-math addition method."
      operationRef = "add_numbers"
    }
  }