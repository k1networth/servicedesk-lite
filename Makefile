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
