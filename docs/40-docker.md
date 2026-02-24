# Docker

Документ фиксирует контейнеризацию сервисов и принципы "prod-friendly" образов.

## Что уже сделано (Iteration 2)
- Multi-stage build (golang builder → минимальный runtime)
- Runtime image: distroless (static-debian12) + запуск под non-root
- Конфигурация через env
- Экспорт портов:
  - ticket-service: 8080
  - outbox-relay: 9090 (метрики)
  - notification-service: 9091 (метрики)

## Состав compose-стека
- Postgres 16
- Kafka (apache/kafka)
- migrate/migrate для миграций
- 3 сервиса приложения (ticket-service, outbox-relay, notification-service)

## Примечания по shutdown
- Для Kubernetes/production важно корректно обрабатывать SIGTERM:
  - ticket-service: перестать принимать новые запросы, завершить активные
  - consumers/relay: остановить polling, завершить текущую обработку

План работ по Kubernetes/HA см. docs/TODO.md.
