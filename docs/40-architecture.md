# Архитектура

## Компоненты
- **ticket-service** — синхронное HTTP API
- **Postgres** — источник истины
- **outbox-relay** — асинхронная доставка событий
- **Kafka** — транспорт событий
- **notification-service** — consumer + идемпотентность

## Transactional outbox
При создании тикета сервис в одной транзакции пишет:
1) строку в `tickets`
2) строку в `outbox`

Это гарантирует: событие публикуется только для реально закоммиченного тикета.

## Статусы outbox
- `pending` — создано, ожидает доставки
- `processing` — забрано relay (через `FOR UPDATE SKIP LOCKED`)
- `sent` — успешно опубликовано в Kafka

Если relay упал во время обработки:
- записи, «застрявшие» в `processing` дольше `processing_timeout`, возвращаются в `pending`.

## Event envelope (Kafka message)
Сообщение в Kafka — JSON envelope:

- `event_id` (UUID) — **ключ идемпотентности**
- `event_type` — например `ticket.created`
- `occurred_at` — RFC3339 timestamp
- `aggregate` — `ticket`
- `aggregate_id` — id тикета
- `request_id` — корреляция/трассировка (опционально)
- `payload` — минимальный набор полей тикета

## Идемпотентный consumer
`notification-service` пишет в `processed_events(event_id)`:
- первый раз: фиксирует `event_id` и выполняет обработку
- повтор: видит `event_id` и безопасно пропускает

Это обеспечивает корректную работу при at-least-once доставке из Kafka.
