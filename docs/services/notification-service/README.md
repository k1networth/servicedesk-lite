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
- `METRICS_ADDR` (по умолчанию `:9091`)

## Идемпотентность
- вставка в `processed_events` по `event_id` (unique)
- повторы безопасно пропускаются
