# OpenUdon Workflow Plan

- Workflow: `openrpc_simple_math_addition`
- Summary: Run the reviewed simple-math addition method using a package-local OpenRPC artifact.
- Version: `openudon.workflow-plan.v1`

## Steps

- `add_numbers` runtime `http` operation `addition`
  - binding: `a <- inputs.left_number`
  - binding: `b <- inputs.right_number`
