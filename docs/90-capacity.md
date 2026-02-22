# Capacity planning & tuning — TODO (чтобы “держать высокий RPS”)

Это не про код, а про инженерные решения и настройки.

## 1) Ticket-service (stateless)
TODO:
- [ ] Параметры runtime:
  - GOMAXPROCS (обычно = CPU limits)
  - http server timeouts
  - max open conns to Postgres (pgxpool max conns)
- [ ] Rate limiting (опционально):
  - per-user или per-ip
- [ ] Concurrency limits:
  - ограничить параллельность тяжелых операций (attachments)

## 2) Postgres
TODO:
- [ ] Индексы как выше (иначе list убьёт БД)
- [ ] Анализ запросов:
  - EXPLAIN (ANALYZE, BUFFERS)
- [ ] VACUUM/Autovacuum (для write heavy)
- [ ] Connection pooling:
  - pgbouncer
- [ ] Read scaling:
  - read replicas (если появится)
  - split read/write (опционально)

## 3) Kafka
TODO:
- [ ] partitions и ключ:
  - больше партиций = больше параллелизма consumer group
  - key = ticket_id (order per ticket)
- [ ] producer batching:
  - linger.ms, batch.size (если актуально в выбранной либе)
- [ ] consumer tuning:
  - max.poll.records, fetch.min.bytes (зависит от либы)
- [ ] ISR/min.insync.replicas/acks=all для HA

## 4) Redis
TODO:
- [ ] cache hit ratio цель (например 60-90% для list)
- [ ] TTL стратегия
- [ ] hot keys (избегать)
- [ ] distributed cache invalidation стратегия (простая и безопасная)

## 5) Outbox relay
TODO:
- [ ] batch size подобрать (например 100-500)
- [ ] poll interval (200-500ms)
- [ ] outbox lag алерты (если растет — relay не справляется)
- [ ] горизонтальный scale relay (несколько реплик) + SKIP LOCKED

## 6) K8s autoscaling
TODO:
- [ ] ticket-service HPA:
  - CPU target (например 60%)
  - optional: custom metric RPS
- [ ] notification-service HPA:
  - лучше по Kafka lag (custom metric) или по CPU + лаг алерт
- [ ] PDB:
  - minAvailable / maxUnavailable
- [ ] topology spread / anti-affinity:
  - чтобы реплики не сидели на одной ноде
