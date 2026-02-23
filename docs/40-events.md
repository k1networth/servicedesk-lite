# Events

События публикуются в Kafka в виде envelope (JSON), чтобы:
- обеспечить слабую связанность сервисов
- упростить отладку
- иметь единый idempotency key (`event_id`)

## Формат
См. `internal/shared/events.Envelope`.

`request_id` прокидывается из HTTP запроса в payload (ticket-service) и поднимается в envelope (outbox-relay) для трассировки.

Ключ сообщения (Kafka key): `aggregate_id`.

## Гарантии
- доставка: at-least-once
- порядок: по ключу (в пределах одного тикета)
- consumer обязан быть идемпотентным (таблица `processed_events`)
