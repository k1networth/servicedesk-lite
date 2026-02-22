SHELL := /usr/bin/env bash

# ---- Tooling (pinned versions, no @latest) ----
BIN_DIR := $(CURDIR)/bin

GOLANGCI_LINT_VERSION := v2.10.1
XTOOLS_VERSION := v0.42.0

GOLANGCI_LINT := $(BIN_DIR)/golangci-lint
GOIMPORTS := $(BIN_DIR)/goimports
OPENAPI_TICKET := api/openapi/ticket-service.yaml

GO ?= go

# Find Go files (skip vendor/bin/.git)
GOFILES := $(shell find . -type f -name '*.go' \
	-not -path './vendor/*' \
	-not -path './bin/*' \
	-not -path './.git/*' 2>/dev/null)

.PHONY: help tools fmt lint test check tidy download clean versions openapi-lint openapi-lint-ticket

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

versions: ## Print pinned tool versions
	@echo "golangci-lint: $(GOLANGCI_LINT_VERSION)"
	@echo "x/tools (goimports): $(XTOOLS_VERSION)"

tools: $(GOLANGCI_LINT) $(GOIMPORTS) ## Install dev tools into ./bin

$(GOLANGCI_LINT):
	@mkdir -p $(BIN_DIR)
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) -> $(GOLANGCI_LINT)"
	@GOBIN=$(BIN_DIR) $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(GOIMPORTS):
	@mkdir -p $(BIN_DIR)
	@echo "Installing goimports (x/tools) $(XTOOLS_VERSION) -> $(GOIMPORTS)"
	@GOBIN=$(BIN_DIR) $(GO) install golang.org/x/tools/cmd/goimports@$(XTOOLS_VERSION)

fmt: tools ## Format code (goimports)
	@if [[ -z "$(GOFILES)" ]]; then \
		echo "No .go files found (nothing to format)."; \
		exit 1; \
	fi
	@$(GOIMPORTS) -w $(GOFILES)

lint: tools ## Run linter
	@$(GOLANGCI_LINT) run -c .golangci.yml ./...

test: ## Run tests
	@$(GO) test ./... -count=1

check: fmt lint test ## Run fmt + lint + test

tidy: ## go mod tidy
	@$(GO) mod tidy

download: ## go mod download
	@$(GO) mod download

clean: ## Remove local tools (./bin)
	@rm -rf $(BIN_DIR)

openapi-lint: openapi-lint-ticket ## Lint OpenAPI specs (docker)

openapi-lint-ticket:
	@docker run --rm -v $(CURDIR):/work -w /work stoplight/spectral:6 lint $(OPENAPI_TICKET)