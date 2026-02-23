# Services

Документация по конкретным сервисам.

- `ticket-service/README.md` — HTTP API тикетов (health/ready, создание, получение)
- `outbox-relay/README.md` — доставка событий из outbox в Kafka
- `notification-service/README.md` — Kafka consumer + идемпотентность

## Конвенции

Каждый сервисный README содержит:
1) Назначение
2) Как запустить локально
3) Конфигурация (ENV)
4) API (или ссылка на OpenAPI)
5) Зависимости (DB/Kafka/Redis)
6) Метрики/логи (минимум)
7) Статус (что готово / что дальше)
