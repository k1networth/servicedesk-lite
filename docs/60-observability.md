# Observability — TODO (golden signals + autoscaling signals)

## Метрики
TODO:
- [ ] HTTP: RPS, latency, errors
- [ ] Outbox lag
- [ ] Kafka consumer lag
- [ ] DB pool stats (опционально)
- [ ] Redis hit/miss (опционально)

## Логи
TODO:
- [ ] JSON logs with request_id/trace_id
- [ ] error logs with stack/context

## Трейсинг
TODO:
- [ ] OTel, propagate trace context
- [ ] correlate logs <-> traces

## Алёрты (минимум)
TODO:
- [ ] 5xx rate
- [ ] P95 latency above threshold
- [ ] outbox lag > X seconds
- [ ] consumer lag > Y messages
