# servicedesk-lite

Минимальный backend «службы поддержки» для демонстрации связки **Transactional Outbox + Kafka + Idempotent Consumer**
в контексте диплома про Kubernetes/DevOps: миграции, метрики, health/ready, скрипты проверки.

## Сервисы

- **ticket-service** (`cmd/ticket-service`)
  - REST API: создать/получить тикет
  - Пишет в Postgres и добавляет событие в outbox **в одной транзакции**
  - Метрики + health/ready

- **outbox-relay** (`cmd/outbox-relay`)
  - Опрашивает таблицу `outbox` (claim через `FOR UPDATE SKIP LOCKED`)
  - Публикует события в Kafka
  - Обновляет статусы outbox: `pending -> processing -> sent`
  - Возвращает «зависшие» события из `processing` обратно в `pending` по таймауту
  - После `OUTBOX_RELAY_MAX_ATTEMPTS` переводит событие в `failed` (и опционально публикует в DLQ)

- **notification-service** (`cmd/notification-service`)
  - Kafka consumer (group по умолчанию: `notification-service`)
  - Таблица `processed_events` для **идемпотентности**
  - Дубликаты по `event_id` безопасно скипаются
  - После `NOTIFY_MAX_ATTEMPTS` переводит событие в `failed` (и опционально публикует в DLQ)

## Архитектура (E2E)

`POST /tickets`
→ транзакция БД: insert в `tickets` + insert в `outbox`
→ outbox-relay публикует событие в Kafka
→ notification-service потребляет и фиксирует обработку в `processed_events` (идемпотентно)

## Быстрый старт (рекомендуется)

### Требования
- Docker + docker compose
- Go
- `make`
- `python3` (скрипты)
- `jq` (опционально)

### 1) Настройка env
Скопируй `.env.example` → `.env` и проверь ключевые параметры:

**Важно:** для Postgres используй `127.0.0.1`, а не `localhost` (иначе возможны проблемы с IPv6 `::1`).

Пример:
- `DATABASE_URL=postgres://servicedesk:servicedesk@127.0.0.1:5432/servicedesk?sslmode=disable`
- `KAFKA_BROKERS=localhost:29092`
- `KAFKA_TOPIC=tickets.events`
- `KAFKA_START_OFFSET=last` — для нового consumer group без commit offset: `first` или `last`

Дополнительно (по умолчанию всё работает и без этого):
- `KAFKA_DLQ_TOPIC=tickets.dlq` — если задан, outbox-relay и notification-service будут писать «непоправимые» события в DLQ
- `OUTBOX_RELAY_MAX_ATTEMPTS=10` — лимит попыток публикации из outbox
- `NOTIFY_MAX_ATTEMPTS=10` — лимит попыток обработки сообщения консьюмером

### 2) E2E-проверка одной командой
Скрипт:
- поднимает Postgres + Kafka (docker compose)
- применяет миграции
- запускает 3 сервиса (ticket/relay/notify)
- создаёт тикет
- проверяет: outbox `sent`, Kafka содержит `event_id`, `processed_events` = `done`

```bash
./scripts/e2e_local.sh
```

### 2a) Полный стек в контейнерах (compose core)

Если нужно запускать **всё как набор контейнеров** (без `go run`), используй compose core:

```bash
make up
make e2e-core
make down
```

То же самое напрямую скриптом:

```bash
./scripts/e2e_compose.sh
```

Демо-сценарий (показать надежность на защите):
```bash
# 1) принудительный фейл consumer → retries → processed_events=failed + DLQ
# 2) остановка Kafka → retries relay → outbox=failed
./scripts/e2e_local.sh --demo

# выборочно
./scripts/e2e_local.sh --demo-notify
./scripts/e2e_local.sh --demo-outbox
```

### 3) Диагностика
```bash
./scripts/diag.sh topics
./scripts/diag.sh peek tickets.events
./scripts/diag.sh outbox
./scripts/diag.sh processed
./scripts/diag.sh group notification-service
```

## Ручной запуск (3 терминала)

```bash
# infra
docker compose --env-file .env -f infra/local/compose.yaml up -d
make migrate-up

# терминал 1
make run-ticket

# терминал 2
make run-relay

# терминал 3
make run-notify
```

Создать тикет:
```bash
curl -i -X POST http://localhost:8080/tickets   -H 'Content-Type: application/json'   -H 'X-Request-Id: demo-1'   -d '{"title":"Printer is broken","description":"Office 3rd floor"}'
```

## Порты

- ticket-service: `:8080`
  - `/healthz`, `/readyz`, `/metrics`
- outbox-relay metrics: `:9090/metrics`
- notification-service metrics: `:9091/metrics`
- Postgres: `:5432`
- Kafka host listener: `:29092`

## Make targets

- `make check` — fmt + lint + test
- `make db-up` / `make db-down` — локальная инфраструктура (compose)
- `make up` / `make down` / `make logs` — полный стек (postgres+kafka+3 сервиса) в контейнерах
- `make migrate-up` — миграции
- `make run-ticket` / `make run-relay` / `make run-notify` — запуск сервисов
- `make e2e` — E2E (deps в docker, сервисы через `go run`)
- `make e2e-core` — E2E по compose core (всё в контейнерах)

## Заметки

- Kafka настроена с двумя listener’ами:
  - внутри docker-сети: `kafka:9092`
  - с хоста (Go-сервисы): `localhost:29092`
- Не используй не-ASCII символы в `KAFKA_TOPIC` (например, кириллическую `с`).
