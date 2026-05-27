.PHONY: help test vet check apitools-boundary readiness release-check release-saas-check release-eval eval-seed-build icot-authoring-scorecard icot-variants-validate product-smoke-check product-smoke-live siblings validate-uws eval synthesize-support build-support promote-support assess-support

GO ?= go
OPENUDON_LLM_PROVIDER ?= copilot-api
OPENUDON_LLM_MODEL ?= gpt-5.4-mini
OPENUDON_RELEASE_MIN_BRIEFS ?= $(shell find ./examples/eval -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')
OPENUDON_RELEASE_SITE_DIR ?= /tmp/openudon-mkdocs-release
OPENUDON_RELEASE_DEMO_ROOT ?= .openudon-run/release-saas-check
OPENUDON_RELEASE_SAAS_FIXTURES ?= slack-message-audit-log gmail-send-audit-receipt itops-slack-jira-issue-intake itops-incident-response-archive order-fulfillment-chain weather-toronto
OPENUDON_RELEASE_DEMO_FIXTURES ?= gmail-send-audit-receipt order-fulfillment-chain

help:
	@echo "Targets: test, vet, check, readiness, release-check, release-saas-check, release-eval, eval-seed-build, icot-authoring-scorecard, icot-variants-validate, product-smoke-check, product-smoke-live, siblings, validate-uws, eval, synthesize-support, build-support, promote-support, assess-support"

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

check: test siblings apitools-boundary

apitools-boundary:
	$(GO) run ./cmd/openudon check-apitools-boundary

readiness:
	$(GO) run ./cmd/openudon readiness --run-gates --out eval/readiness/local.json

release-check:
	$(GO) test ./...
	$(GO) vet ./...
	$(MAKE) check
	git diff --check

release-saas-check:
	$(MAKE) release-check
	$(MAKE) eval-seed-build
	$(MAKE) icot-variants-validate
	$(MAKE) icot-authoring-scorecard
	$(MAKE) validate-uws
	$(GO) run ./cmd/openudon check-doc-memory
	$(GO) run ./cmd/openudon n8n-bridge validate --root examples/eval
	mkdocs build --strict --site-dir $(OPENUDON_RELEASE_SITE_DIR)
	for fixture in $(OPENUDON_RELEASE_SAAS_FIXTURES); do \
		$(GO) run ./cmd/icot lint --example ./examples/eval/$$fixture; \
	done
	rm -rf "$(OPENUDON_RELEASE_DEMO_ROOT)"
	mkdir -p "$(OPENUDON_RELEASE_DEMO_ROOT)/approvals"
	for fixture in $(OPENUDON_RELEASE_DEMO_FIXTURES); do \
		cp -R "examples/eval/$$fixture" "$(OPENUDON_RELEASE_DEMO_ROOT)/$$fixture"; \
		mkdir -p "$(OPENUDON_RELEASE_DEMO_ROOT)/$$fixture/workflows"; \
		cp "$(OPENUDON_RELEASE_DEMO_ROOT)/$$fixture/reference/intent.hcl" "$(OPENUDON_RELEASE_DEMO_ROOT)/$$fixture/workflows/intent.hcl"; \
		$(GO) run ./cmd/openudon build --example "$(OPENUDON_RELEASE_DEMO_ROOT)/$$fixture"; \
		$(GO) run ./cmd/openudon assess --example "$(OPENUDON_RELEASE_DEMO_ROOT)/$$fixture"; \
		$(GO) run ./cmd/openudon approval-template --example "$(OPENUDON_RELEASE_DEMO_ROOT)/$$fixture" --state approved_for_sandbox --reviewer "Release SaaS Check" > "$(OPENUDON_RELEASE_DEMO_ROOT)/approvals/$$fixture.json"; \
		$(GO) run ./cmd/openudon run --example "$(OPENUDON_RELEASE_DEMO_ROOT)/$$fixture" --tier sandbox --approval "$(OPENUDON_RELEASE_DEMO_ROOT)/approvals/$$fixture.json" --workdir "$(OPENUDON_RELEASE_DEMO_ROOT)/workdir-$$fixture" --dry-run; \
	done

release-eval:
	$(GO) run ./cmd/openudon eval --root ./examples/eval --provider $(OPENUDON_LLM_PROVIDER) --model $(OPENUDON_LLM_MODEL) --release-gate --min-briefs $(OPENUDON_RELEASE_MIN_BRIEFS)

eval-seed-build:
	$(GO) test ./internal/icot -run TestEvalReferenceSeedBuildMatrix -count=1

icot-authoring-scorecard:
	$(GO) run ./cmd/icot scorecard --root examples/eval --include-variants --out eval/runs/icot-authoring-scorecard-local

icot-variants-validate:
	$(GO) run ./cmd/icot variants validate --root examples/eval

product-smoke-check:
	$(GO) run ./cmd/openudon smoke-matrix --mode dry-run --workdir .openudon-run/product-smoke --out .openudon-run/product-smoke/summary.json

product-smoke-live:
	$(GO) run ./cmd/openudon smoke-matrix --mode live --workdir .openudon-run/product-smoke --out .openudon-run/product-smoke/summary.json

siblings:
	$(GO) run ./cmd/openudon check

validate-uws:
	$(GO) run ./cmd/openudon validate ./examples/uws-validation

eval:
	$(GO) run ./cmd/openudon eval --root ./examples/eval --provider $(OPENUDON_LLM_PROVIDER) --model $(OPENUDON_LLM_MODEL)

synthesize-support:
	$(GO) run ./cmd/openudon synthesize --example ./examples/support-email --provider $(OPENUDON_LLM_PROVIDER) --model $(OPENUDON_LLM_MODEL)

build-support:
	$(GO) run ./cmd/openudon build --example ./examples/support-email --provider $(OPENUDON_LLM_PROVIDER) --model $(OPENUDON_LLM_MODEL)

promote-support:
	$(GO) run ./cmd/openudon promote --example ./examples/support-email

assess-support:
	$(GO) run ./cmd/openudon assess --example ./examples/support-email
