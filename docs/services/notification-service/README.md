# notification-service

Kafka consumer, демонстрирующий идемпотентную обработку через БД.

## Запуск
```
make run-notify
```

## Env
- `DATABASE_URL` (обязательно)
- `KAFKA_BROKERS` (при запуске на хосте: `localhost:29092`)
- `KAFKA_TOPIC` (по умолчанию `tickets.events`)
- `KAFKA_GROUP_ID` (по умолчанию `notification-service`)
- `KAFKA_START_OFFSET` (по умолчанию `last`) — для нового consumer group: `first` или `last`
- `METRICS_ADDR` (по умолчанию `:9091`)
- `KAFKA_DLQ_TOPIC` (опционально) — писать непоправимые события в DLQ
- `NOTIFY_MAX_ATTEMPTS` (по умолчанию `10`) — лимит попыток обработки одного сообщения
- `NOTIFY_FORCE_FAIL` (опционально) — принудительно фейлить обработку (для демо/тестов)
- `NOTIFY_FORCE_FAIL_EVENT_TYPE` (опционально) — фейлить только указанный `event_type` (по умолчанию все)

## Идемпотентность
- вставка в `processed_events` по `event_id` (unique)
- повторы безопасно пропускаются
