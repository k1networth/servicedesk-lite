# Testing

## Что уже есть

### Unit tests
- Базовые unit-тесты есть (например, внутренние пакеты с логикой тикетов).
- Расширение unit coverage — опционально (не блокирует диплом), но можно добавить точечно.

### E2E (Local)
Проверяет happy-path: ticket → outbox → kafka → consumer → processed_events.

Команды:
- `make e2e`

### E2E (Compose core)
То же самое, но против полностью контейнеризированного стека (без `go run`).

Команды:
- `make up`
- `make e2e-core`
- `make down`

### Resilience demos (Iteration 1)
Демо-сценарии для защиты (наблюдаемость + надёжность):
- `./scripts/e2e_local.sh --demo-notify` — retries → processed_events=failed → DLQ
- `./scripts/e2e_local.sh --demo-outbox` — Kafka down → outbox=failed + метрики

## Что можно добавить позже (не блокирует текущую стадию)
- Нагрузочные тесты (k6/vegeta) + замеры P95/P99
- Chaos/resilience сценарии в k8s (kill pod, проверка lag/метрик)

План работ по нагрузке/хаосу — docs/TODO.md.
