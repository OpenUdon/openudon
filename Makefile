.PHONY: help test vet check doc-memory apitools-boundary readiness release-check release-eval siblings validate-uws eval synthesize-support build-support promote-support assess-support run-example

GO ?= go
RAMEN_PROVIDER ?= copilot-api
RAMEN_MODEL ?= gpt-5.4-mini
RAMEN_RELEASE_MIN_BRIEFS ?= $(shell find ./examples/eval -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')

help:
	@echo "Targets: test, vet, check, doc-memory, readiness, release-check, release-eval, siblings, validate-uws, eval, synthesize-support, build-support, promote-support, assess-support, run-example"

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

check: test siblings apitools-boundary
	$(GO) run ./cmd/ramen check

doc-memory:
	./scripts/check-doc-memory.sh

apitools-boundary:
	./scripts/check-apitools-boundary.sh

readiness:
	$(GO) run ./cmd/ramen readiness --run-gates --out eval/readiness/local.json

release-check:
	$(GO) test ./...
	$(GO) vet ./...
	$(MAKE) check
	git diff --check

release-eval:
	$(GO) run ./cmd/ramen eval --root ./examples/eval --provider $(RAMEN_PROVIDER) --model $(RAMEN_MODEL) --release-gate --min-briefs $(RAMEN_RELEASE_MIN_BRIEFS)

siblings:
	./scripts/check-siblings.sh

validate-uws:
	./scripts/validate-uws.sh ./examples/support-email/workflows

eval:
	$(GO) run ./cmd/ramen eval --root ./examples/eval --provider $(RAMEN_PROVIDER) --model $(RAMEN_MODEL)

synthesize-support:
	$(GO) run ./cmd/ramen synthesize --example ./examples/support-email --provider $(RAMEN_PROVIDER) --model $(RAMEN_MODEL)

build-support:
	$(GO) run ./cmd/ramen build --example ./examples/support-email --provider $(RAMEN_PROVIDER) --model $(RAMEN_MODEL)

promote-support:
	$(GO) run ./cmd/ramen promote --example ./examples/support-email

assess-support:
	$(GO) run ./cmd/ramen assess --example ./examples/support-email

run-example:
	./scripts/run-udon.sh ./examples/support-email/workflows/workflow.uws.yaml ./examples/support-email
