# NFR — нефункциональные требования (High RPS + HA)

Этот документ нужен, чтобы проект выглядел инженерно: есть SLO, семантики доставки, принципы масштабирования и наблюдаемости.

## 1) Термины
- **RPS** — requests per second.
- **P95/P99 latency** — задержка, ниже которой укладываются 95%/99% запросов.
- **SLI** — метрика качества (например, доля 2xx/3xx или P95 latency).
- **SLO** — целевое значение SLI (например, availability 99.9%).
- **SLA** — контракт с последствиями (в дипломе обычно не требуется).
- **Availability** — доступность сервиса.
- **Error budget** — допустимая доля ошибок/простоя, вытекает из SLO.
- **HA** — высокая доступность (переживаем падение части компонентов).
- **DR** — катастрофоустойчивость (восстановление после больших аварий, обычно отдельная тема).

## 2) Целевые SLO (реалистичные для диплома)
- Ticket API availability: **99.9%/месяц**.
- P95 latency:
  - GET /tickets: **< 200 ms** при 200 RPS
  - POST /tickets: **< 300 ms** при 50 RPS
- Error rate: **< 0.1%** 5xx.
- Async pipeline: "ticket created" → "event published" (outbox → Kafka): **P99 < 2 s**.

> Числа можно уточнить после нагрузочного теста, но важно, что цели зафиксированы.

## 3) Модель нагрузки
- 80% read (GET/list), 20% write (create/update).
- List чаще, чем get.
- Attachments редкие (например, 1%).

## 4) Принципы для высокого RPS
- Stateless сервисы → горизонтальное масштабирование.
- Pooling соединений к Postgres (pgxpool): лимиты + timeouts.
- Timeouts/cancellation везде через context.
- Backpressure:
  - ограничение concurrency на handler/usecase уровне
  - защита от thundering herd (кэш/дедупликация — опционально)
- Pagination:
  - на старте offset/limit
  - при росте — переход на cursor-based pagination (планируется)

## 5) HA принципы
- Рестарт пода не ломает корректность (idempotency на consumer стороне).
- Rolling updates без даунтайма:
  - readiness gates
  - graceful shutdown
- Для k8s: PDB + topology spread/anti-affinity (планируется в Iteration 3).

## 6) Семантика доставки и консистентность
- Sync path (HTTP): ticket created → 201 после коммита в БД.
- Async path (Kafka): **at-least-once**.
- Idempotency consumer: таблица `processed_events` защищает от дублей.
- Exactly-once не требуется: проще и надёжнее at-least-once + idempotency.

## 7) Capacity (в общих чертах)
- Postgres: индексы + анализ запросов + autovacuum.
- Kafka: партиции, ключ (ticket_id), lag алерты.
- K8s: requests/limits + HPA targets.

Детали по capacity и k8s — docs/90-capacity.md и docs/TODO.md.
