# Capacity planning & tuning

Это не про код, а про инженерные решения и настройки. Часть пунктов можно уточнить после нагрузочных тестов.

## 1) Ticket-service (stateless)
Рекомендуемые настройки:
- HTTP server timeouts (read/write/idle)
- Ограничение concurrency для тяжёлых операций (если появятся attachments)
- Rate limiting (опционально)

## 2) Postgres
- Индексы для ключевых запросов (особенно list/get)
- Анализ запросов: EXPLAIN (ANALYZE, BUFFERS)
- Autovacuum/VACUUM (особенно при write-heavy)
- Connection pooling (pgxpool; pgbouncer — опционально)
- Read scaling (реплики) — опционально

## 3) Kafka
- Партиции и ключ:
  - больше партиций → больше параллелизма consumer group
  - key = ticket_id (order per ticket)
- Lag мониторинг и алерты
- HA параметры (prod): replication, min.insync.replicas, acks=all

## 4) Outbox relay
- batch size (например 100–500)
- poll interval (200–500ms)
- outbox lag алерты
- горизонтальный scale relay (несколько реплик) + SKIP LOCKED

## 5) K8s autoscaling (Iteration 3)
- ticket-service HPA по CPU (например target 60%)
- PDB + topology spread

Детальная реализация autoscaling/HA — docs/70-k8s.md и docs/TODO.md.
