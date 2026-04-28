.PHONY: help test vet check release-check release-eval siblings validate-uws eval synthesize-support build-support promote-support assess-support run-example

GO ?= go
RAMEN_PROVIDER ?= gemini
RAMEN_MODEL ?= gemini-2.5-flash

help:
	@echo "Targets: test, vet, check, release-check, release-eval, siblings, validate-uws, eval, synthesize-support, build-support, promote-support, assess-support, run-example"

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

check: test siblings
	$(GO) run ./cmd/ramen check

release-check:
	$(GO) test ./...
	$(GO) vet ./...
	$(MAKE) check
	git diff --check

release-eval:
	$(GO) run ./cmd/ramen eval --root ./examples/eval --provider $(RAMEN_PROVIDER) --model $(RAMEN_MODEL) --release-gate

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
