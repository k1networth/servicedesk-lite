# Архитектура

Проект состоит из двух слоёв: **инфраструктурного** (основной предмет диплома) и **прикладного** (референсная нагрузка для демонстрации инфраструктуры).

---

## Слои инфраструктуры

```
┌─────────────────────────────────────────────────────┐
│  CI/CD Layer                                        │
│  GitHub Actions: lint → test → build → deploy       │
├─────────────────────────────────────────────────────┤
│  Cloud / IaC Layer                                  │
│  Terraform: VPC, Managed K8s, Managed Postgres,     │
│  Container Registry (Yandex Cloud blueprint)        │
├─────────────────────────────────────────────────────┤
│  Observability Layer                                │
│  Prometheus + Grafana (kube-prometheus-stack)       │
│  ServiceMonitors, Dashboards                        │
├─────────────────────────────────────────────────────┤
│  Kubernetes Layer                                   │
│  Helm chart: Deployments, StatefulSet, Ingress,     │
│  HPA, PDB, ConfigMap, Secret, ServiceMonitors       │
├─────────────────────────────────────────────────────┤
│  Container Layer                                    │
│  Multi-stage Docker builds, distroless runtime,     │
│  non-root, docker compose для локальной среды       │
├─────────────────────────────────────────────────────┤
│  Application Layer (референсная нагрузка)           │
│  ticket-service / outbox-relay / notification-svc   │
└─────────────────────────────────────────────────────┘
```

### Container Layer
- Multi-stage build: `golang:1.24-alpine` → `gcr.io/distroless/static-debian12`
- Runtime под non-root пользователем (UID 65532)
- Конфигурация исключительно через переменные окружения (12-factor app)
- Локальная среда: `docker compose` с профилями (`core`, `obs`)

### Kubernetes Layer
Реализован Helm chart (`infra/k8s/helm/servicedesk-lite`).

| Ресурс | Назначение |
|---|---|
| Deployment × 3 | ticket-service, outbox-relay, notification-service |
| StatefulSet × 2 | Postgres (с PVC), Kafka (KRaft mode) |
| Job | Database migrations (при каждом деплое) |
| Service × 5 | ClusterIP для каждого компонента |
| Ingress | Внешний доступ к ticket-service и UI мониторинга |
| HPA | Автомасштабирование ticket-service по CPU (min 1, max 3) |
| PDB | Гарантия minAvailable при rolling update |
| ConfigMap | Нечувствительная конфигурация |
| Secret | Credentials (DATABASE_URL, пароли) |
| ServiceMonitor × 3 | Интеграция с Prometheus |

### Observability Layer
- **Prometheus** — сбор метрик через ServiceMonitor CRD
- **Grafana** — дашборды: RPS/latency, outbox lag, notify processed
- Доступ через Ingress (`/grafana`, `/prometheus`) без port-forward
- Установка одной командой: `make k8s-obs-install && make k8s-obs-apply`

### CI/CD Layer
Описан в [docs/85-cicd.md](85-cicd.md). Реализация — GitHub Actions.

### Cloud / IaC Layer
Production blueprint на Terraform (Yandex Cloud) описан в [docs/80-iac.md](80-iac.md).

---

## Прикладной слой (референсная нагрузка)

**servicedesk-lite** — минимальный backend «службы поддержки», реализующий паттерн **Transactional Outbox** для надёжной асинхронной доставки событий.

### Компоненты

| Сервис | Роль |
|---|---|
| **ticket-service** | HTTP API: создать / получить тикет; пишет в Postgres + outbox в одной транзакции |
| **outbox-relay** | Опрашивает `outbox`, публикует события в Kafka; claim через `FOR UPDATE SKIP LOCKED` |
| **notification-service** | Kafka consumer; идемпотентная обработка через `processed_events` |

### Поток данных (E2E)

```
POST /tickets
  └─► [ticket-service]
        └─► BEGIN TRANSACTION
              ├─► INSERT INTO tickets
              └─► INSERT INTO outbox (status=pending)
            COMMIT ──► 201 Created

[outbox-relay] (polling loop)
  └─► SELECT FOR UPDATE SKIP LOCKED (pending → processing)
  └─► Publish → Kafka: tickets.events
  └─► UPDATE outbox SET status=sent

[notification-service] (consumer group)
  └─► Consume message
  └─► INSERT INTO processed_events (event_id) ON CONFLICT DO NOTHING
  └─► Handle (idempotent)
```

### Transactional Outbox

Запись в `tickets` и `outbox` происходит в **одной транзакции** — событие публикуется только для реально закоммиченного тикета. Это устраняет split-brain между БД и брокером без двухфазного коммита.

### Event Envelope (Kafka message)

| Поле | Описание |
|---|---|
| `event_id` | UUID — ключ идемпотентности |
| `event_type` | `ticket.created` |
| `occurred_at` | RFC3339 timestamp |
| `aggregate` | `ticket` |
| `aggregate_id` | ID тикета (Kafka message key) |
| `request_id` | Корреляция/трассировка |
| `payload` | Поля тикета |

### Гарантии доставки

| Путь | Семантика |
|---|---|
| HTTP (sync) | Ответ 201 только после коммита в БД |
| Kafka (async) | At-least-once |
| Consumer | Идемпотентен (защита через `processed_events`) |

### Статусы outbox

| Статус | Описание |
|---|---|
| `pending` | Создано, ожидает доставки |
| `processing` | Забрано relay (`FOR UPDATE SKIP LOCKED`) |
| `sent` | Успешно опубликовано в Kafka |
| `failed` | Превышен лимит попыток (→ DLQ, если настроен) |

Если relay упал во время обработки: записи, «застрявшие» в `processing` дольше `processing_timeout`, возвращаются в `pending` автоматически.
