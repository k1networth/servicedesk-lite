# outbox-relay

Опрашивает таблицу outbox в Postgres и публикует события в Kafka.

## Запуск
```
make run-relay
```

## Env
- `DATABASE_URL` (обязательно)
- `KAFKA_BROKERS` (при запуске на хосте: `localhost:29092`)
- `KAFKA_TOPIC` (по умолчанию `tickets.events`)
- `POLL_INTERVAL` (по умолчанию `500ms`)
- `BATCH_SIZE` (по умолчанию `50`)
- `OUTBOX_PROCESSING_TIMEOUT` (по умолчанию `30s`)
- `METRICS_ADDR` (по умолчанию `:9090`)

## Поведение
- claim `pending` событий через `FOR UPDATE SKIP LOCKED`
- выставляет `processing_started_at`
- публикует в Kafka
- помечает `sent` и ставит `sent_at`
- возвращает зависшие `processing` обратно в `pending`
