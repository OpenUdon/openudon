# Tutorial: Order Fulfillment Chain

This fixture is the multi-service SaaS release demo. It fetches customer and
inventory records, then creates a sandbox fulfillment order from reviewed
request mappings. It demonstrates cross-service data flow, mixed credential
binding names, side-effect review, approval JSON, and trusted-runner dry-run
evidence.

Fixture path:

```text
examples/eval/order-fulfillment-chain/project.md
examples/eval/order-fulfillment-chain/openapi/customers.yaml
examples/eval/order-fulfillment-chain/openapi/inventory.yaml
examples/eval/order-fulfillment-chain/openapi/orders.yaml
examples/eval/order-fulfillment-chain/reference/intent.hcl
```

## Provider-Free Artifact Loop

For release evidence, run this tutorial in an ignored workdir and build from
the checked-in reference intent:

```bash
export DEMO_ROOT=.openudon-run/order-fulfillment-tutorial
rm -rf "$DEMO_ROOT"
mkdir -p "$DEMO_ROOT/approvals"
cp -R examples/eval/order-fulfillment-chain "$DEMO_ROOT/order-fulfillment-chain"
mkdir -p "$DEMO_ROOT/order-fulfillment-chain/workflows"
cp "$DEMO_ROOT/order-fulfillment-chain/reference/intent.hcl" \
  "$DEMO_ROOT/order-fulfillment-chain/workflows/intent.hcl"

go run ./cmd/openudon build --example "$DEMO_ROOT/order-fulfillment-chain"
go run ./cmd/openudon assess --example "$DEMO_ROOT/order-fulfillment-chain"
```

Inspect:

```text
.openudon-run/order-fulfillment-tutorial/order-fulfillment-chain/expected/plan.md
.openudon-run/order-fulfillment-tutorial/order-fulfillment-chain/expected/quality.md
.openudon-run/order-fulfillment-tutorial/order-fulfillment-chain/expected/review.md
.openudon-run/order-fulfillment-tutorial/order-fulfillment-chain/expected/symphony-handoff.json
```

The plan should show `get_customer` and `check_inventory` feeding
`create_fulfillment_order`. Review evidence should list `customers_bearer_token`,
`inventory_api_key`, and `orders_bearer_token` as symbolic bindings only.

## Approval Dry Run

Create sandbox approval from the current ignored workdir package, then dry-run
the trusted handoff:

```bash
go run ./cmd/openudon approval-template \
  --example "$DEMO_ROOT/order-fulfillment-chain" \
  --state approved_for_sandbox \
  --reviewer "Reviewer Name" \
  > "$DEMO_ROOT/approvals/order-fulfillment-chain.json"

go run ./cmd/openudon run \
  --example "$DEMO_ROOT/order-fulfillment-chain" \
  --tier sandbox \
  --approval "$DEMO_ROOT/approvals/order-fulfillment-chain.json" \
  --workdir "$DEMO_ROOT/workdir-order-fulfillment-chain" \
  --dry-run
```

The dry run checks approval, digest, quality, handoff policy, credential-value
policy, and tier compatibility. It does not contact customers, inventory,
fulfillment, n8n, Terraform/OpenTofu, Symphony, or udon.

Remove `--dry-run` only in a trusted operator environment with sandbox targets,
reviewed credential bindings, and a configured `OPENUDON_EXECUTOR`.
