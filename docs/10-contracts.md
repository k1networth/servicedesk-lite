# Contracts / Spec — API + Kafka + DB + Redis + Metrics (RPS/HA oriented) — TODO

Здесь “спецификация” проекта. Реализация должна ей соответствовать.

---

## A) HTTP API (ticket-service)
Источник истины (OpenAPI): `api/openapi/ticket-service.yaml`  
Формат ошибок: `{ error: { code, message, request_id } }`



### A0. Общие требования (прод-ориентиры)
TODO:
- [ ] Стандартные таймауты:
  - server read header timeout
  - request context timeout (на уровне middleware) (например 2-5s)
- [ ] Ограничения:
  - max request body size (например 1MB для JSON)
  - rate limiting (опционально; можно сделать позже)
- [ ] Request ID:
  - вход: X-Request-Id
  - выход: X-Request-Id
- [ ] Auth (упрощенно):
  - X-User-Id обязателен для бизнес ручек
  - 401 если отсутствует
- [ ] Единый формат ошибок (json + request_id)
- [ ] Версионирование API (опционально):
  - /v1/tickets ... (можно сразу заложить)

### A1. GET /healthz
TODO:
- [ ] 200 если процесс жив
- [ ] не делать тяжелых проверок

### A2. GET /readyz
TODO:
- [ ] 200 если готов принимать трафик
- [ ] проверки зависимостей (по мере появления):
  - Postgres: SELECT 1
  - Redis: PING
  - Kafka: metadata fetch / ping admin (опционально)
- [ ] если зависимость недоступна -> 503

### A3. POST /tickets
TODO:
Request:
- URL: /tickets
- Headers:
  - X-User-Id
  - Idempotency-Key (опционально)
- Body: {"title":"...","description":"..."}
Validation:
- title 3..200
- description 1..5000

Response:
- 201 + ticket json

DB (важно для RPS/HA):
- [ ] операция должна быть **быстрой**:
  - 1 insert tickets
  - 1 insert outbox_events
- [ ] строго в одной транзакции

Outbox:
- event_type ticket.created
- payload минимум: ticket_id, created_by, title, request_id

Idempotency (Redis):
- key: idem:<user_id>:<idempotency_key>
- ttl: 1h
- поведение:
  - если key найден -> вернуть тот же ticket (или тот же ticket_id)
  - если не найден -> создать ticket -> записать key

### A4. GET /tickets/{id}
TODO:
- 200 + ticket
- 404 если нет
- 403 если тикет чужой (реши и зафиксируй)

### A5. GET /tickets
TODO:
- query: status, limit, offset
- defaults: limit=50, offset=0
- max limit=200
- 200 + array tickets
Performance notes:
- [ ] индекс (created_by, status, created_at desc)
- [ ] избегать тяжелых сортировок без индекса
- [ ] кэшировать ответ (Redis) на короткий ttl
Redis cache:
- key: tickets:list:<user>:<status>:<limit>:<offset>
- ttl: 60s
- invalidation:
  - при create/close тикета -> delete relevant keys (можно “грубо”: delete pattern по user)

### A6. POST /tickets/{id}/close
TODO:
- статус CLOSED
- response: 200 updated ticket или 204
- идемпотентность:
  - вариант 1: если уже CLOSED -> 200/204 (предпочтительнее для high RPS)
  - вариант 2: 409 already_closed
Outbox:
- ticket.closed

### A7. Attachments (S3/MinIO)
TODO:
- POST /tickets/{id}/attachments (multipart)
- GET /tickets/{id}/attachments
Performance notes:
- [ ] uploads ограничить по размеру
- [ ] хранить только метаданные в Postgres
- [ ] выдавать presigned URL (опционально) чтобы разгрузить API

---

## B) Kafka (HA friendly)

### B1. Топики
TODO:
- tickets.events (основной)
- tickets.dlq (dead-letter)

### B2. Envelope
Value JSON:
- event_id (uuid)
- event_type
- created_at
- payload
Key:
- ticket_id (order per ticket)

### B3. HA и настройки (прод вариант)
TODO (для доков/диплома, локально можно проще):
- [ ] replication factor = 3 (в прод кластере)
- [ ] min.insync.replicas = 2
- [ ] producer acks=all
- [ ] enable idempotent producer (если библиотека позволяет)
- [ ] partitions:
  - выбрать число партиций под throughput и параллелизм consumer’ов
- [ ] consumer group scaling:
  - максимальная параллельность = partitions

### B4. Consumer semantics
TODO:
- at-least-once
- commit offset после успешной обработки
- retry/backoff
- DLQ после N попыток

---

## C) Postgres (HA friendly)

### C1. Таблицы
TODO: tickets, outbox_events, attachments, notifications, processed_events

### C2. Индексы (критично для list)
TODO:
- tickets(created_by, created_at desc)
- tickets(created_by, status, created_at desc)
- outbox_events(published_at, created_at)

### C3. HA варианты (описать в дипломе)
TODO:
- Managed Postgres (cloud) + read replicas
- Self-managed:
  - Patroni + etcd (или оператор в k8s: CloudNativePG / Zalando)
- Connection pooling:
  - pgbouncer (в k8s sidecar/service)
- Migrations:
  - отдельный job в k8s перед rollout

---

## D) Transactional Outbox (scale-out relay)
TODO:
- SQL fetch: FOR UPDATE SKIP LOCKED
- relay масштабируется горизонтально (несколько реплик)
- published_at ставится только после publish success
- метрика outbox_lag_seconds

---

## E) Redis (HA friendly)
TODO:
- локально: single redis
- прод: Sentinel или Redis Cluster
- политики:
  - TTL, maxmemory-policy (allkeys-lru или volatile-lru)
- ключи:
  - cache keys для list
  - idem keys
- защита от stampede:
  - singleflight/locking (опционально)

---

## F) Metrics (для HPA/alerts)
TODO:
- HTTP RPS и latency (golden signals)
- Kafka lag (ключевой для autoscaling consumer)
- outbox lag (ключевой для здоровья pipeline)
