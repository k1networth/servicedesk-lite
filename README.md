# servicedesk-lite

ServiceDesk-lite — учебный (pet) проект для ВКР: **простая доменная область (тикеты)**, но реализация и инфраструктура — как в продакшене: **микросервисы**, **Kubernetes**, **DevOps**, **observability**, подготовка к **high RPS** и **HA**.

## Быстрые ссылки

- Документация: `docs/README.md`
- Контракты/спеки: `docs/10-contracts.md`
- Исходники сервиса: `cmd/ticket-service`, `internal/ticket`

## Goals

- Построить “production-ready” каркас микросервисов на Go
- Обкатать полный цикл: форматирование/линт/тесты → контейнеризация → CI/CD → деплой в Kubernetes
- Добавить наблюдаемость: метрики/логи/трейсы (Prometheus/Grafana + ELK/Loki)
- Подготовить архитектуру к масштабированию: stateless сервисы, HPA, очереди, кэш, outbox, идемпотентность
- Описать и обосновать экономический эффект/выгоды (скорость доставки, надежность, автоматизация)

## High-level architecture (planned)

Сервисы (простые по функционалу, “взрослые” по инженерии):

- **ticket-service** — HTTP API для тикетов (сейчас in-memory; далее Postgres + Redis)
- **outbox-relay** — Transactional Outbox → публикация событий в Kafka (scale-out)
- **notification-service** — Kafka consumer → уведомления (идемпотентно, DLQ)

Зависимости (эволюционно):

- **Postgres** (миграции; далее HA/оператор или managed)
- **Kafka** (топики/партиции; далее 3 брокера + replication)
- **Redis** (cache + idempotency; далее Sentinel/Cluster)
- **S3/MinIO** (attachments)
- **Observability**: Prometheus/Grafana + logs (ELK или Loki) + tracing (OTel)

## Repo structure

- `cmd/` — точки входа сервисов
- `internal/` — бизнес-логика и адаптеры
- `api/` — контракты (OpenAPI и др.)
- `infra/` — docker-compose/k8s/terraform/ansible (по итерациям)
- `build/` — Dockerfile и сборка (по итерациям)
- `docs/` — требования/контракты/дизайн/итерации

## Dev setup (Linux)

### Requirements

- Go 1.22+
- git
- make

### Quick start

Установить dev tools (версии прибиты в Makefile, ставятся в `./bin`):

```bash
make tools
```

Форматирование / линт / тесты:

```bash
make fmt
make lint
make test
```

Или всё сразу:

```bash
make check
```

Очистить локальные тулзы:

```bash
make clean
```

### Запуск ticket-service локально

```bash
go run ./cmd/ticket-service
```

По умолчанию слушает `:8080` (см. `internal/shared/config` и `.env`).

Полезные эндпоинты:

- `GET /healthz`
- `GET /readyz`
- `POST /tickets`
- `GET /tickets/{id}`

> Контракт этих эндпоинтов фиксируется в документации и (на следующей итерации) в OpenAPI спецификации.

## Iterations / Roadmap

Подход: итерации с “production requirements” (наблюдаемость, graceful shutdown, конфиг, тестируемость).

Статус (на сейчас):

1. **Bootstrap**: go.mod + Makefile (fmt/lint/test/tools) + базовые стандарты — ✅
2. **ticket-service skeleton**: health/ready, request-id, structured logs, graceful shutdown — ✅
3. **Postgres**: миграции + CRUD тикетов — ⏳ (после фиксации API контракта)
4. **Kafka base**: топики, producer/consumer, семантика at-least-once — ⏳
5. **Transactional Outbox**: relay scale-out (SKIP LOCKED), метрики lag — ⏳
6. **OpenAPI контракт**: `api/openapi/ticket-service.yaml` + простая проверка спеки — 🔜
7. **Redis**: cache + idempotency-key, политика TTL/invalidation — ⏳
8. **Observability**: Prometheus/Grafana + логи (ELK/Loki) + (опц.) tracing — ⏳
9. **Kubernetes**: Helm, HPA/PDB/anti-affinity, rollout без даунтайма — ⏳
10. **HA dependencies**: Kafka 3 brokers, Redis HA, Postgres HA/оператор (или managed) + демо отказоустойчивости — ⏳

## License

MIT — см. `LICENSE`.
