# ticket-service

HTTP API для управления тикетами.

## Status


OpenAPI: `api/openapi/ticket-service.yaml`

Сейчас:
- `GET /healthz`, `GET /readyz`
- `POST /tickets` (in-memory)
- `GET /tickets/{id}` (in-memory)

Дальше:
- Итерация 6: OpenAPI контракт `api/openapi/ticket-service.yaml`
- Затем: Postgres + миграции, идемпотентность, метрики, интеграционные тесты

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
