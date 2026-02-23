# Контракты

## HTTP API (ticket-service)

Base: `http://localhost:8080`

- `POST /tickets`
  - создаёт тикет
  - возвращает `201` и JSON тикета

- `GET /tickets/{id}`
  - возвращает JSON тикета или `404`

- `GET /healthz` / `GET /readyz`
- `GET /metrics` (Prometheus)

## Kafka

Topic: `tickets.events` (env `KAFKA_TOPIC`)

### Envelope (JSON)
Поля:
- `event_id` (string, UUID)
- `event_type` (string)
- `occurred_at` (string, RFC3339)
- `aggregate` (string)
- `aggregate_id` (string)
- `request_id` (string, optional)
- `payload` (object)

## База данных

Таблицы:
- `tickets`
- `outbox`
- `processed_events` (идемпотентность consumer’а)
