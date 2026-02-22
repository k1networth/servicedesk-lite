SHELL := /usr/bin/env bash

# ---- Tooling (pinned versions, no @latest) ----
BIN_DIR := $(CURDIR)/bin

GOLANGCI_LINT_VERSION := v2.10.1
XTOOLS_VERSION := v0.42.0

GOLANGCI_LINT := $(BIN_DIR)/golangci-lint
GOIMPORTS := $(BIN_DIR)/goimports

GO ?= go

# Find Go files (skip vendor/bin/.git)
GOFILES := $(shell find . -type f -name '*.go' \
	-not -path './vendor/*' \
	-not -path './bin/*' \
	-not -path './.git/*' 2>/dev/null)

.PHONY: help tools fmt lint test check tidy download clean versions

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

# TODO: Напиши Makefile с нуля (без копипасты).
# Цель: удобные команды dev/ci/release для **3 сервисов** и **инфры**.
#
# 1) Базовые цели
# - help: вывести список команд (awk/grep по комментариям)
# - fmt: gofmt + goimports (если используешь)
# - lint: golangci-lint
# - test: unit tests
# - test-integration: integration tests (поднимают Postgres/Redis/Kafka/MinIO)
# - build: собрать бинарники (linux/amd64 + linux/arm64 опционально)
#
# 2) Инфра (локально)
# - dc-up-core: поднять postgres+redis+kafka (минимум)
# - dc-up-observability: добавить prometheus+grafana+loki/elk
# - dc-up-storage: добавить minio
# - dc-down: остановить и почистить volumes (осторожно)
#
# 3) Миграции
# - migrate-up / migrate-down / migrate-create
# - важно: миграции должны запускаться в CI и в init job k8s
#
# 4) Docker
# - docker-build: build images for each service
# - docker-push: push images
# - tag стратегия: git sha + semver
#
# 5) K8s
# - k3d-up / k3d-down
# - helm-install / helm-upgrade
# - rollout-status / rollback
#
# 6) Load tests
# - load-smoke / load-rps
# - профилирование pprof (если добавишь)
#
# TODO: продумай переменные
# - SERVICE=ticket-service|outbox-relay|notification-service
# - ENV=local|dev|stage
# - TAG=$(git rev-parse --short HEAD)
# - KUBECONTEXT=...
