# Воспроизведение демо

Три сценария: локальный, Kubernetes (kind), CI/CD.

---

## Требования

| Инструмент | Версия | Зачем |
|---|---|---|
| Go | 1.24+ | сборка и тесты |
| Docker + compose | 24+ | локальная инфраструктура |
| kind | 0.23+ | локальный k8s кластер |
| kubectl | 1.30+ | управление кластером |
| helm | 3.15+ | деплой Helm chart |
| make | — | единая точка входа |

---

## Сценарий 1 — Локально (compose)

```bash
# 1. Склонировать и настроить env
git clone https://github.com/k1networth/servicedesk-lite
cd servicedesk-lite
cp .env.example .env          # DATABASE_URL=postgres://servicedesk:servicedesk@127.0.0.1:5432/servicedesk?sslmode=disable

# 2. Поднять Postgres + Kafka, применить миграции, запустить сервисы и прогнать E2E
./scripts/e2e_local.sh

# Ожидаемый результат:
# ✓ ticket created (id=...)
# ✓ outbox status=sent
# ✓ kafka message received (event_id=...)
# ✓ processed_events status=done
```

Или вручную (полный стек в контейнерах):

```bash
make up          # docker compose: postgres + kafka + все сервисы
make e2e-core    # E2E проверка против compose-стека
make down        # остановить
```

---

## Сценарий 2 — Kubernetes (kind)

```bash
# 1. Создать кластер, задеплоить стек (одна команда)
make k8s-up

# Что происходит внутри:
# - kind create cluster --config infra/k8s/kind/kind-config.yaml
# - kubectl apply -f infra/k8s/addons/ingress-nginx/
# - docker build всех образов с тегом :dev
# - kind load docker-image (загрузить образы в кластер)
# - helm upgrade --install servicedesk-lite ...
# - kubectl wait --for=condition=ready pod ...

# 2. Проверить API
curl -s -X POST http://localhost:8080/tickets \
  -H 'Content-Type: application/json' \
  -H 'X-Request-Id: demo-001' \
  -d '{"title":"k8s demo","description":"testing k8s deployment"}'
# → {"id":"...","title":"k8s demo","status":"open"}

curl -s http://localhost:8080/tickets/<id>
# → {"id":"...","title":"k8s demo","status":"open"}

# 3. Проверить readiness
curl -s http://localhost:8080/readyz   # → ready
curl -s http://localhost:8080/healthz  # → ok

# 4. Установить мониторинг
make k8s-obs-install   # kube-prometheus-stack в namespace observability
make k8s-obs-apply     # ServiceMonitors + Grafana dashboard

# 5. Нагрузочный тест (нужен hey: go install github.com/rakyll/hey@latest)
hey -n 100000 -c 200 \
  -m POST \
  -H 'Content-Type: application/json' \
  -d '{"title":"load test","description":"perf"}' \
  http://localhost:8080/tickets
# Ожидаемый результат: ~1800 RPS, 0 ошибок

# 6. Открыть Grafana
make k8s-obs-ui
# http://localhost:8080/grafana  (admin / admin)

# 7. Удалить кластер
make k8s-down
```

### Что показать на защите

| Шаг | Что демонстрирует |
|---|---|
| `make k8s-up` | Воспроизводимый деплой одной командой |
| `kubectl get pods -n servicedesk` | Все поды Running |
| `curl POST /tickets` | Работающий API |
| `kubectl get hpa -n servicedesk` | HPA настроен |
| Grafana dashboard | Observability: RPS, latency, outbox lag |
| `hey` нагрузочный тест | ~1800 RPS, 0% ошибок |

---

## Сценарий 3 — CI/CD (GitHub Actions)

```bash
# Пуш в main запускает пайплайн автоматически
git push origin main

# Джобы в Actions:
# ✓ Lint   — go vet + golangci-lint
# ✓ Test   — go test -race
# ✓ Build  — go build ./cmd/...
# ✓ Vuln   — govulncheck
# ✓ Images — push в GHCR (только main)
#   ghcr.io/k1networth/servicedesk/ticket-service:latest
#   ghcr.io/k1networth/servicedesk/outbox-relay:latest
#   ghcr.io/k1networth/servicedesk/notification-service:latest
#   ghcr.io/k1networth/servicedesk/migrate:latest
```

Результат виден на: `https://github.com/k1networth/servicedesk-lite/actions`

---

## Полезные команды

```bash
make help          # список всех команд

# Разработка
make run-ticket    # запустить ticket-service локально
make run-relay     # запустить outbox-relay локально
make run-notify    # запустить notification-service локально

# Тесты
make test          # go test ./...
make test-race     # go test -race ./...

# Качество
make lint          # golangci-lint
make vet           # go vet
make check         # fmt + vet + lint + test

# K8s
make k8s-up        # создать кластер + задеплоить
make k8s-down      # удалить кластер
make k8s-status    # kubectl get all -n servicedesk
make k8s-logs      # логи всех сервисов

# Мониторинг
make k8s-obs-install
make k8s-obs-apply
make k8s-obs-ui
```
