# NFR — нефункциональные требования (High RPS + HA) — TODO

Этот документ нужен, чтобы проект выглядел “взросло” и чтобы ты не писал хаосом.

## 1) Термины
TODO:
- [ ] RPS: requests per second
- [ ] P95/P99 latency
- [ ] SLI/SLO/SLA (что измеряем и какие цели)
- [ ] Availability (доступность), Error budget
- [ ] HA vs DR (высокая доступность vs катастрофоустойчивость)

## 2) Целевые SLO (для диплома можно задать реалистично)
TODO: выбрать и зафиксировать цифры (примерные ориентиры)
- [ ] Ticket API availability: 99.9% (за месяц)
- [ ] P95 latency для GET /tickets: < 200ms при N RPS
- [ ] P95 latency для POST /tickets: < 300ms при N RPS
- [ ] Error rate: < 0.1% 5xx
- [ ] Async pipeline: “event published to Kafka” в пределах 2s (P99)

## 3) Трафик и нагрузка (модель)
TODO:
- [ ] Опиши типичный паттерн:
  - 80% reads (GET/list), 20% writes (create/close)
  - list чаще, чем get
  - attachments реже (например 1%)
- [ ] Пиковая нагрузка:
  - N RPS steady, M RPS peak (например 200/1000 для демо)
- [ ] Размеры payload:
  - title/description ограничения
  - attachments max size (например 10MB)

## 4) Архитектурные принципы для высокого RPS
TODO:
- [ ] stateless сервисы (горизонтальное масштабирование)
- [ ] connection pooling (Postgres: pgxpool + лимиты)
- [ ] timeouts и cancellation (context deadlines)
- [ ] backpressure:
  - ограничение concurrency на handler/usecase уровне
  - защита от “thundering herd” (кэш + singleflight опционально)
- [ ] caching:
  - list endpoints кэшировать кратко (60s)
  - ETag/If-None-Match (опционально)
- [ ] pagination:
  - offset pagination на старте
  - TODO: миграция на cursor-based pagination, если надо

## 5) HA принципы (что должно переживать)
TODO:
- [ ] Под рестартом пода сервис не теряет корректность (idempotency)
- [ ] Потеря 1 ноды Kafka/Redis/Postgres не должна останавливать систему (в прод варианте)
- [ ] Rolling updates без даунтайма:
  - readiness gates
  - graceful shutdown (дрейнить запросы, остановить consumer корректно)
- [ ] PDB и anti-affinity в k8s

## 6) Data consistency и семантики доставки
TODO:
- [ ] Sync path (HTTP): ticket created -> 201 после коммита БД
- [ ] Async path (Kafka):
  - at-least-once
  - idempotent consumer через processed_events
- [ ] exactly-once: не обязателен, но опиши почему выбрал at-least-once + idempotency

## 7) Capacity “на пальцах” (для защиты)
TODO:
- [ ] Postgres: R/W throughput, индексы, VACUUM
- [ ] Kafka: partitions, replication, throughput, lag
- [ ] Redis: memory sizing, eviction policy
- [ ] K8s: requests/limits, HPA targets
