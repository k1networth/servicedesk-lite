# servicedesk-lite

ServiceDesk-lite — учебный (pet) проект для ВКР: **простая доменная область (тикеты)**, но реализация и инфраструктура — как в продакшене: **микросервисы**, **Kubernetes**, **DevOps**, **observability**, подготовка к **high RPS** и **HA** (Kafka/Postgres/Redis кластера, autoscaling).

## Goals
- Построить “production-ready” каркас микросервисов на Go
- Обкатать полный цикл: форматирование/линт/тесты → контейнеризация → CI/CD → деплой в Kubernetes
- Добавить наблюдаемость: метрики/логи/трейсы (Prometheus/Grafana + ELK/Loki)
- Подготовить архитектуру к масштабированию: stateless сервисы, HPA, очереди, кэш, outbox, идемпотентность
- Описать и обосновать экономический эффект/выгоды (время деплоя, надежность, автоматизация)

## High-level architecture (planned)
Сервисы (простые по функционалу, “взрослые” по инженерии):
- **ticket-service** — HTTP API для тикетов (Postgres + Redis)
- **outbox-relay** — Transactional Outbox → публикация событий в Kafka (scale-out)
- **notification-service** — Kafka consumer → уведомления (идемпотентно, DLQ)

Зависимости (эволюционно):
- **Postgres** (с миграциями; далее HA/оператор или managed)
- **Kafka** (с топиками/партициями; далее 3 брокера + replication)
- **Redis** (cache + idempotency; далее Sentinel/Cluster)
- **S3/MinIO** (attachments)
- **Observability**: Prometheus/Grafana + logs (ELK или Loki) + tracing (OTel)

## Repo structure
- `cmd/` — точки входа сервисов
- `internal/` — бизнес-логика и адаптеры
- `api/` — OpenAPI/контракты (позже)
- `infra/` — docker-compose/k8s/terraform/ansible (по итерациям)
- `build/` — Dockerfile и сборка (позже)
- `docs/` — заметки/дизайн/итерации (позже можно расширять)

## Dev setup (Linux)

### Requirements
- Go 1.22+
- git
- make

### Quick start
Установить dev tools (версии прибиты в Makefile, ставятся в `./bin`):
```bash
make tools
````

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

## Iterations / Roadmap

Подход: итерации с “production requirements” (наблюдаемость, graceful shutdown, конфиг, тестируемость).

Примерный план:

1. **Bootstrap**: go.mod + Makefile (fmt/lint/test/tools) + базовые стандарты
2. **ticket-service skeleton**: health/ready/metrics, request-id, structured logs, graceful shutdown
3. **Postgres**: миграции + CRUD тикетов, индексы под list
4. **Kafka base**: топики, producer/consumer, семантика at-least-once
5. **Transactional Outbox**: relay scale-out (SKIP LOCKED), метрики lag
6. **Redis**: cache + idempotency-key, политика TTL/invalidation
7. **Observability**: Prometheus/Grafana + логи (ELK/Loki) + (опц.) tracing
8. **Kubernetes**: Helm, HPA/PDB/anti-affinity, rollout без даунтайма
9. **HA dependencies**: Kafka 3 brokers, Redis HA, Postgres HA/оператор (или managed) + демо отказоустойчивости

## License

MIT — см. `LICENSE`.
