package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genelet/udon/pkg/rollout"
)

func TestXRD004OpenAPIFixtureCoverage(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "eval")

	cursor := parseReferenceIntent(t, root, "cursor-pagination-report")
	cursorStep := findStep(cursor, "list_events_second_page")
	if cursorStep == nil {
		t.Fatal("cursor fixture missing second page step")
	}
	if cursorStep.Operation != "listAuditEvents" {
		t.Fatalf("second page operation = %q, want listAuditEvents", cursorStep.Operation)
	}
	if got := bindSource(cursorStep, "list_events_first_page", "cursor"); got != "received_body.page.nextCursor" {
		t.Fatalf("cursor binding = %q, want received_body.page.nextCursor", got)
	}
	if got := cursorStep.With["Authorization"]; got != "audit_events_bearer_token" {
		t.Fatalf("cursor fixture bearer binding = %q", got)
	}

	chain := parseReferenceIntent(t, root, "order-fulfillment-chain")
	create := findStep(chain, "create_fulfillment_order")
	if create == nil {
		t.Fatal("fulfillment fixture missing create_fulfillment_order step")
	}
	if create.OpenAPI != "openapi/orders.yaml" || create.Operation != "createFulfillmentOrder" {
		t.Fatalf("create step target = %s %s", create.OpenAPI, create.Operation)
	}
	if got := bindSource(create, "get_customer", "shippingAddressId"); got != "received_body.defaultShippingAddress.id" {
		t.Fatalf("shipping address binding = %q", got)
	}
	if got := bindSource(create, "check_inventory", "warehouseId"); got != "received_body.preferredWarehouseId" {
		t.Fatalf("warehouse binding = %q", got)
	}
	if len(distinctStepOpenAPIs(chain)) < 3 {
		t.Fatalf("fulfillment fixture should use at least three step-local OpenAPI documents")
	}

	assertFixtureFileContains(t, root, "order-fulfillment-chain", filepath.Join("openapi", "orders.yaml"), "requestBody:", "post:", "securitySchemes:")
	assertFixtureFileContains(t, root, "cursor-pagination-report", filepath.Join("openapi", "audit-events.yaml"), "securitySchemes:", "nextCursor:")
}

func parseReferenceIntent(t *testing.T, root, name string) *rollout.Intent {
	t.Helper()
	intent, err := rollout.ParseIntentFile(filepath.Join(root, name, "reference", "intent.hcl"))
	if err != nil {
		t.Fatalf("parse %s reference intent: %v", name, err)
	}
	return intent
}

func findStep(intent *rollout.Intent, name string) *rollout.Step {
	for _, step := range flattenSteps(intent) {
		if step != nil && step.Name == name {
			return step
		}
	}
	return nil
}

func flattenSteps(intent *rollout.Intent) []*rollout.Step {
	if intent == nil {
		return nil
	}
	var out []*rollout.Step
	var walk func([]*rollout.Step)
	walk = func(steps []*rollout.Step) {
		for _, step := range steps {
			if step == nil {
				continue
			}
			out = append(out, step)
			walk(step.Steps)
			for _, branch := range step.Cases {
				if branch != nil {
					walk(branch.Steps)
				}
			}
			if step.Default != nil {
				walk(step.Default.Steps)
			}
		}
	}
	walk(intent.Steps)
	return out
}

func bindSource(step *rollout.Step, from, target string) string {
	if step == nil {
		return ""
	}
	for _, bind := range step.Binds {
		if bind != nil && bind.From == from {
			return bind.Fields[target]
		}
	}
	return ""
}

func distinctStepOpenAPIs(intent *rollout.Intent) map[string]bool {
	out := map[string]bool{}
	for _, step := range flattenSteps(intent) {
		if step != nil && strings.TrimSpace(step.OpenAPI) != "" {
			out[strings.TrimSpace(step.OpenAPI)] = true
		}
	}
	return out
}

func assertFixtureFileContains(t *testing.T, root, fixture, rel string, needles ...string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, fixture, rel))
	if err != nil {
		t.Fatalf("read %s/%s: %v", fixture, rel, err)
	}
	text := string(data)
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			t.Fatalf("%s/%s missing %q", fixture, rel, needle)
		}
	}
}
