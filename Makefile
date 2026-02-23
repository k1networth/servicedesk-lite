SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c
.DEFAULT_GOAL := help

BIN_DIR := $(CURDIR)/bin

GOLANGCI_LINT_VERSION := v2.10.1
XTOOLS_VERSION := v0.42.0

GOLANGCI_LINT := $(BIN_DIR)/golangci-lint
GOIMPORTS := $(BIN_DIR)/goimports

OPENAPI_TICKET := api/openapi/ticket-service.yaml
DB_COMPOSE := infra/local/compose.yaml

ENV_FILE := $(CURDIR)/.env
-include $(ENV_FILE)
export

GOFILES := $(shell git ls-files '*.go' 2>/dev/null)

WORKDIR_MOUNT := -v $(CURDIR):/work -w /work
DOCKER_RUN := docker run --rm $(WORKDIR_MOUNT)

MIGRATE_IMAGE := migrate/migrate:v4.17.1
MIGRATE_NET ?= host
MIGRATE := $(DOCKER_RUN) --network $(MIGRATE_NET) $(MIGRATE_IMAGE)

GO_TEST_FLAGS ?= -count=1

.PHONY: help versions tools fmt lint test check tidy download clean
.PHONY: openapi-lint openapi-lint-ticket
.PHONY: db-up db-down db-ps migrate-up migrate-down guard-%

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

versions: ## Print pinned tool versions
	@echo "golangci-lint: $(GOLANGCI_LINT_VERSION)"
	@echo "x/tools (goimports): $(XTOOLS_VERSION)"
	@echo "migrate: $(MIGRATE_IMAGE)"

tools: $(GOLANGCI_LINT) $(GOIMPORTS) ## Install dev tools into ./bin

$(GOLANGCI_LINT):
	@mkdir -p $(BIN_DIR)
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) -> $(GOLANGCI_LINT)"
	@GOBIN=$(BIN_DIR) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(GOIMPORTS):
	@mkdir -p $(BIN_DIR)
	@echo "Installing goimports (x/tools) $(XTOOLS_VERSION) -> $(GOIMPORTS)"
	@GOBIN=$(BIN_DIR) go install golang.org/x/tools/cmd/goimports@$(XTOOLS_VERSION)

fmt: tools ## Format code (goimports)
	@if [[ -z "$(GOFILES)" ]]; then echo "No .go files found (skip)."; exit 0; fi
	@$(GOIMPORTS) -w $(GOFILES)

lint: tools ## Run linter
	@$(GOLANGCI_LINT) run -c .golangci.yml ./...

test: ## Run tests
	@go test ./... $(GO_TEST_FLAGS)

check: fmt lint test ## Run fmt + lint + test

tidy: ## go mod tidy
	@go mod tidy

download: ## go mod download
	@go mod download

clean: ## Remove local tools (./bin)
	@rm -rf $(BIN_DIR)

openapi-lint: openapi-lint-ticket ## Lint OpenAPI specs (docker)

openapi-lint-ticket:
	@$(DOCKER_RUN) stoplight/spectral:6 lint $(OPENAPI_TICKET)

db-up: ## Start local dependencies via docker compose
	@docker compose --env-file $(ENV_FILE) -f $(DB_COMPOSE) up -d

db-down: ## Stop local dependencies (and remove volumes)
	@docker compose --env-file $(ENV_FILE) -f $(DB_COMPOSE) down -v

db-ps: ## Show local dependencies status
	@docker compose --env-file $(ENV_FILE) -f $(DB_COMPOSE) ps

guard-%:
	@if [[ -z "$($*)" ]]; then echo "ERROR: $* is empty"; exit 1; fi

migrate-up: guard-DATABASE_URL ## Run DB migrations up (docker)
	@$(MIGRATE) -path migrations -database "$(DATABASE_URL)" up

migrate-down: guard-DATABASE_URL ## Rollback last migration (docker)
	@$(MIGRATE) -path migrations -database "$(DATABASE_URL)" down 1