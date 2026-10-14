SHELL := /bin/bash

.PHONY: all init test coverage test-live lint vet contracts sam-models help

all: help

# the release tooling (semantic-release, commitlint), and the pre-commit hooks.
init:
	npm install --include=dev
	pre-commit install

test:
	go test -race ./...

# fails below 90% coverage of the smarter package.
coverage:
	go test -race -coverprofile=coverage.out .
	@go tool cover -func=coverage.out | tail -1
	@go tool cover -func=coverage.out | tail -1 | awk '{ sub(/%/, "", $$3); if ($$3 + 0 < 90) { print "coverage is below 90%"; exit 1 } }'

# Test the SDK against a running Smarter platform, e.g. the local one:
# SMARTER_BASE_URL=http://localhost:9357 SMARTER_API_KEY=... make test-live
# These prompt an LLMClient, which makes billable LLM requests.
test-live:
	SMARTER_TEST_LIVE=true go test -count=1 -run Live -v .

vet:
	go vet ./...
	golangci-lint run ./...

lint:
	pre-commit run --all-files

# Copy the platform's api contracts from ../smarter: the chat and journal contracts, the SAM schemas
# and examples, and the api's OpenAPI document. A ref reads them at a commit or tag, e.g.
# make contracts SMARTER_REF=v0.22.2
contracts:
	scripts/sync-contracts.sh ../smarter $(SMARTER_REF)

# Regenerate sam/ from the platform's committed JSON Schemas.
sam-models: contracts
	go run ./internal/cmd/samgen
	go test ./sam/

help:
	@echo '===================================================================='
	@echo 'init          - install the release tooling and the pre-commit hooks'
	@echo 'test          - run the unit tests, with the race detector'
	@echo 'coverage      - run the unit tests with coverage'
	@echo 'test-live     - test against a running Smarter platform (SMARTER_BASE_URL, SMARTER_API_KEY)'
	@echo 'vet           - go vet and golangci-lint'
	@echo 'lint          - run every pre-commit hook on every file'
	@echo 'contracts     - copy the api contracts, SAM schemas and OpenAPI document from ../smarter'
	@echo 'sam-models    - copy the contracts, and regenerate the sam package from their schemas'
	@echo '===================================================================='
