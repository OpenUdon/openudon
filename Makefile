.PHONY: help test vet check standalone-build public-boundary fast

GO ?= go

help:
	@echo "Targets: test, vet, check, standalone-build, public-boundary, fast"

test:
	GOWORK=off $(GO) test ./...

vet:
	GOWORK=off $(GO) vet ./...

standalone-build:
	GOWORK=off $(GO) build ./...

public-boundary:
	GOWORK=off $(GO) test ./tests/publicapi -count=1

check: standalone-build test vet

fast: test
