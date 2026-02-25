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
CORE_COMPOSE := infra/local/compose.core.yaml

ENV_FILE := $(CURDIR)/.env
-include $(ENV_FILE)
export

# Find Go files (skip vendor/bin/.git). Avoids failures on deleted-but-not-staged tracked files.
GOFILES := $(shell find . -type f -name '*.go' \
	-not -path './vendor/*' \
	-not -path './bin/*' \
	-not -path './.git/*' 2>/dev/null)

WORKDIR_MOUNT := -v $(CURDIR):/work -w /work
DOCKER_RUN := docker run --rm $(WORKDIR_MOUNT)

MIGRATE_IMAGE := migrate/migrate:v4.17.1
MIGRATE_NET ?= host
MIGRATE := $(DOCKER_RUN) --network $(MIGRATE_NET) $(MIGRATE_IMAGE)

GO_TEST_FLAGS ?= -count=1

# diag helpers
TOPIC ?= tickets.events
GROUP ?= notification-service

.PHONY: help versions tools fmt lint test check tidy download clean
.PHONY: openapi-lint openapi-lint-ticket
.PHONY: db-up db-down db-ps db-logs db-reset migrate-up migrate-down guard-%
.PHONY: run-ticket run-relay run-notify e2e e2e-core
.PHONY: up down ps logs
.PHONY: diag diag-topics diag-peek diag-outbox diag-processed diag-groups diag-group

# k8s/kind demo helpers
.PHONY: docker-build kind-up kind-down kind-load k8s-addons kind-demo k8s-install k8s-uninstall k8s-status k8s-port-forward

KIND_CLUSTER_NAME ?= servicedesk
KIND_CONFIG ?= infra/k8s/kind/kind-config.yaml

HELM_RELEASE ?= servicedesk
HELM_NAMESPACE ?= servicedesk
IMAGE_TAG ?= dev

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

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

db-logs: ## Follow docker compose logs
	@docker compose --env-file $(ENV_FILE) -f $(DB_COMPOSE) logs -f --tail=200

db-reset: db-down db-up ## Recreate local dependencies from scratch (with fresh volumes)

up: ## Start full local stack (postgres+kafka+services) via docker compose
	@docker compose --env-file $(ENV_FILE) -f $(CORE_COMPOSE) up -d --build

down: ## Stop full local stack (and remove volumes)
	@docker compose --env-file $(ENV_FILE) -f $(CORE_COMPOSE) down -v

ps: ## Show full local stack status
	@docker compose --env-file $(ENV_FILE) -f $(CORE_COMPOSE) ps

logs: ## Follow full local stack logs
	@docker compose --env-file $(ENV_FILE) -f $(CORE_COMPOSE) logs -f --tail=200

guard-%:
	@if [[ -z "$($*)" ]]; then echo "ERROR: $* is empty"; exit 1; fi

migrate-up: guard-DATABASE_URL ## Run DB migrations up (docker)
	@$(MIGRATE) -path migrations -database "$(DATABASE_URL)" up

migrate-down: guard-DATABASE_URL ## Rollback last migration (docker)
	@$(MIGRATE) -path migrations -database "$(DATABASE_URL)" down 1

run-ticket: ## Run ticket-service (Ctrl+C is OK)
	@go run ./cmd/ticket-service; code=$$?; test $$code -eq 0 -o $$code -eq 130

run-relay: ## Run outbox-relay (Ctrl+C is OK)
	@go run ./cmd/outbox-relay; code=$$?; test $$code -eq 0 -o $$code -eq 130

run-notify: ## Run notification-service (Ctrl+C is OK)
	@go run ./cmd/notification-service; code=$$?; test $$code -eq 0 -o $$code -eq 130

e2e: ## Run local end-to-end check (scripts/e2e_local.sh)
	@./scripts/e2e_local.sh

e2e-core: ## Run end-to-end check against docker compose core stack (no go run)
	@./scripts/e2e_compose.sh

diag: ## Show diag script help
	@./scripts/diag.sh

diag-topics: ## List Kafka topics
	@./scripts/diag.sh topics

diag-peek: ## Peek Kafka messages (TOPIC=tickets.events)
	@./scripts/diag.sh peek $(TOPIC)

diag-outbox: ## Show outbox rows
	@./scripts/diag.sh outbox

diag-processed: ## Show processed_events rows
	@./scripts/diag.sh processed

diag-groups: ## List consumer groups
	@./scripts/diag.sh groups

diag-group: ## Describe consumer group (GROUP=notification-service)
	@./scripts/diag.sh group $(GROUP)

docker-build: ## Build local images for kind/helm demo
	@docker build -t servicedesk/ticket-service:$(IMAGE_TAG) -f cmd/ticket-service/Dockerfile .
	@docker build -t servicedesk/outbox-relay:$(IMAGE_TAG) -f cmd/outbox-relay/Dockerfile .
	@docker build -t servicedesk/notification-service:$(IMAGE_TAG) -f cmd/notification-service/Dockerfile .
	@docker build -t servicedesk/migrate:$(IMAGE_TAG) -f build/migrate/Dockerfile .

kind-up: ## Create kind cluster (requires kind)
	@CLUSTER_NAME=$(KIND_CLUSTER_NAME) CONFIG=$(KIND_CONFIG) bash ./scripts/kind.sh up

kind-down: ## Delete kind cluster
	@CLUSTER_NAME=$(KIND_CLUSTER_NAME) bash ./scripts/kind.sh down

kind-load: ## Load built images into kind cluster
	@kind load docker-image --name $(KIND_CLUSTER_NAME) \
		servicedesk/ticket-service:$(IMAGE_TAG) \
		servicedesk/outbox-relay:$(IMAGE_TAG) \
		servicedesk/notification-service:$(IMAGE_TAG) \
		servicedesk/migrate:$(IMAGE_TAG)

k8s-addons: ## Install ingress-nginx + metrics-server (internet required)
	@# kind single-node prep: allow scheduling on control-plane + mark ingress-ready
	@for n in $$(kubectl get nodes -o name 2>/dev/null); do \
	  kubectl label $$n ingress-ready=true --overwrite >/dev/null 2>&1 || true; \
	  kubectl taint $$n node-role.kubernetes.io/control-plane- >/dev/null 2>&1 || true; \
	  kubectl taint $$n node-role.kubernetes.io/master- >/dev/null 2>&1 || true; \
	done
	@kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.11.3/deploy/static/provider/kind/deploy.yaml
	@kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
	@# kind fix: metrics-server -> kubelet TLS/addressing
	@kubectl -n kube-system patch deploy metrics-server --type='json' -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"},{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname"}]' >/dev/null 2>&1 || true
	@kubectl -n kube-system rollout restart deploy metrics-server >/dev/null 2>&1 || true
	@kubectl -n kube-system rollout status deploy metrics-server --timeout=300s
	@kubectl -n ingress-nginx rollout status deploy/ingress-nginx-controller --timeout=300s
	@kubectl -n kube-system wait --for=condition=Ready pod -l k8s-app=metrics-server --timeout=300s

kind-demo: docker-build kind-up kind-load k8s-addons k8s-install ## One-command kind demo (build->cluster->addons->helm)

k8s-install: ## Install/upgrade Helm release into kind (requires helm+kubectl)
	@helm upgrade --install $(HELM_RELEASE) infra/k8s/helm/servicedesk-lite \
		-n $(HELM_NAMESPACE) --create-namespace \
		--set images.ticketService.tag=$(IMAGE_TAG) \
		--set images.outboxRelay.tag=$(IMAGE_TAG) \
		--set images.notificationService.tag=$(IMAGE_TAG) \
		--set images.migrate.tag=$(IMAGE_TAG)

k8s-uninstall: ## Uninstall Helm release
	@helm uninstall $(HELM_RELEASE) -n $(HELM_NAMESPACE)

k8s-status: ## kubectl get all in namespace
	@kubectl -n $(HELM_NAMESPACE) get all

k8s-port-forward: ## Port-forward ticket-service to localhost:8080
	@kubectl -n $(HELM_NAMESPACE) port-forward svc/ticket-service 8080:8080
