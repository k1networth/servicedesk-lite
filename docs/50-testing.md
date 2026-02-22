# Testing — TODO (включая нагрузку и отказоустойчивость)

## Unit tests
TODO:
- [ ] domain validation
- [ ] usecases orchestration (mock repos)
- [ ] error mapping

## Integration tests
TODO:
- [ ] Postgres: CRUD + outbox rows
- [ ] Kafka: relay publishes, consumer processes
- [ ] Redis: idempotency + cache

## Load tests
TODO:
- [ ] k6/vegeta:
  - GET /tickets at N RPS
  - POST /tickets at M RPS
- [ ] измерить P95/P99

## Chaos/Resilience tests (минимум)
TODO:
- [ ] убить relay -> lag растёт -> после восстановления догоняет
- [ ] убить consumer -> lag растёт -> после восстановления догоняет
- [ ] simulate duplicate events -> consumer idempotent
