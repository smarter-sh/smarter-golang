SHELL := /bin/bash
# pre-commit runs from a Python virtual environment in .venv/, made by make python-init. Its name
# starts with a dot so that go build, vet, test and golangci-lint skip it, as ./... skips dot
# directories: pre-commit ships a .go file, which would otherwise be linted as part of this module.
PYTHON := python3.13
PRE_COMMIT := .venv/bin/pre-commit

.PHONY: all init python-init clean test coverage test-live lint vet contracts sam-models help

all: help

# the release tooling (semantic-release, commitlint), and the pre-commit hooks.
init:
	npm install --include=dev
	$(MAKE) python-init

# Creates .venv/ with pre-commit, and installs the pre-commit and commit-msg hooks. See .pre-commit-config.yaml.
# The Go hooks also need goimports and golangci-lint on PATH.
python-init:
	$(PYTHON) -m venv .venv
	.venv/bin/pip install --upgrade pip
	.venv/bin/pip install -r requirements/local.txt
	$(PRE_COMMIT) install

clean:
	rm -rf node_modules .venv coverage.out

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
	$(PRE_COMMIT) run --all-files

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
	@echo 'python-init   - create .venv/ with pre-commit, and install the pre-commit and commit-msg hooks'
	@echo 'clean         - delete node_modules, .venv/ and coverage.out'
	@echo 'test          - run the unit tests, with the race detector'
	@echo 'coverage      - run the unit tests with coverage'
	@echo 'test-live     - test against a running Smarter platform (SMARTER_BASE_URL, SMARTER_API_KEY)'
	@echo 'vet           - go vet and golangci-lint'
	@echo 'lint          - run every pre-commit hook on every file'
	@echo 'contracts     - copy the api contracts, SAM schemas and OpenAPI document from ../smarter'
	@echo 'sam-models    - copy the contracts, and regenerate the sam package from their schemas'
	@echo '===================================================================='
