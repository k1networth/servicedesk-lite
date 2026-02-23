# Observability

## Метрики
- ticket-service: `http://localhost:8080/metrics`
- outbox-relay: `http://localhost:9090/metrics`
- notification-service: `http://localhost:9091/metrics`

## Рекомендуемые метрики
- ticket-service:
  - requests total (route/status)
  - latency histogram (если добавишь)
- outbox-relay:
  - published total / failed total
  - outbox lag seconds (now - oldest pending)
- notification-service:
  - `notify_processed_total{event_type,status}`

## Корреляция
- Используй заголовок `X-Request-Id` для HTTP.
- `request_id` может прокидываться в Kafka envelope для трассировки.
