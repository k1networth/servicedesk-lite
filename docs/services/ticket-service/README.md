# ticket-service

HTTP API для управления тикетами.

## Status


OpenAPI: `api/openapi/ticket-service.yaml`

Сейчас:
- `GET /healthz`, `GET /readyz` *(пока plain text; будет приведено к JSON по OpenAPI)*
- `POST /tickets` (in-memory или Postgres)
- `GET /tickets/{id}` (in-memory или Postgres)

Дальше:
- Привести реализацию к контракту OpenAPI (health/ready JSON, Location header и т.п.)
- Добавить list/close, индексы, интеграционные тесты
- Затем: идемпотентность (Redis) и outbox-relay

## Run locally

```bash
go run ./cmd/ticket-service
```

По умолчанию сервис слушает `:8080`.

## Endpoints

- `GET /healthz` — liveness
- `GET /readyz` — readiness
- `POST /tickets` — создать тикет
- `GET /tickets/{id}` — получить тикет

## Configuration (ENV)

Минимально (пример; будет расширяться):

- `HTTP_ADDR` — адрес HTTP сервера (например `:8080`)
- `LOG_LEVEL` — debug/info/warn/error
- `DATABASE_URL` — если задан, сервис работает с Postgres (и пишет в outbox в одной транзакции)

## Error format (global)

Единый формат ошибок для сервисов:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "title is required",
    "request_id": "..."
  }
}
```

Подробности кодов/правил — см. `docs/10-contracts.md`.
