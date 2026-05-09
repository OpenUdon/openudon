.PHONY: help test vet check doc-memory apitools-boundary readiness release-check release-eval siblings validate-uws eval synthesize-support build-support promote-support assess-support

GO ?= go
OPENUDON_PROVIDER ?= copilot-api
OPENUDON_MODEL ?= gpt-5.4-mini
OPENUDON_RELEASE_MIN_BRIEFS ?= $(shell find ./examples/eval -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')

help:
	@echo "Targets: test, vet, check, doc-memory, readiness, release-check, release-eval, siblings, validate-uws, eval, synthesize-support, build-support, promote-support, assess-support"

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

check: test siblings apitools-boundary

doc-memory:
	$(GO) run ./cmd/openudon check-doc-memory

apitools-boundary:
	$(GO) run ./cmd/openudon check-apitools-boundary

readiness:
	$(GO) run ./cmd/openudon readiness --run-gates --out eval/readiness/local.json

release-check:
	$(GO) test ./...
	$(GO) vet ./...
	$(MAKE) check
	git diff --check

release-eval:
	$(GO) run ./cmd/openudon eval --root ./examples/eval --provider $(OPENUDON_PROVIDER) --model $(OPENUDON_MODEL) --release-gate --min-briefs $(OPENUDON_RELEASE_MIN_BRIEFS)

siblings:
	$(GO) run ./cmd/openudon check

validate-uws:
	$(GO) run ./cmd/openudon validate ./examples/uws-validation

eval:
	$(GO) run ./cmd/openudon eval --root ./examples/eval --provider $(OPENUDON_PROVIDER) --model $(OPENUDON_MODEL)

synthesize-support:
	$(GO) run ./cmd/openudon synthesize --example ./examples/support-email --provider $(OPENUDON_PROVIDER) --model $(OPENUDON_MODEL)

build-support:
	$(GO) run ./cmd/openudon build --example ./examples/support-email --provider $(OPENUDON_PROVIDER) --model $(OPENUDON_MODEL)

promote-support:
	$(GO) run ./cmd/openudon promote --example ./examples/support-email

assess-support:
	$(GO) run ./cmd/openudon assess --example ./examples/support-email
