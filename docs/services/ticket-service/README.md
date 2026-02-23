# ticket-service

REST API для тикетов.

## Запуск
```
make run-ticket
```

## Env
- `HTTP_ADDR` (по умолчанию `:8080`)
- `DATABASE_URL` (если пустой, реализация может работать in-memory, если это предусмотрено)

## Endpoints
- `POST /tickets`
- `GET /tickets/{id}`
- `/healthz`, `/readyz`, `/metrics`
