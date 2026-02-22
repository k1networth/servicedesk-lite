# Local dev (compose) — TODO (под high-RPS/HA)

Локальная среда должна помогать разработке, но не превращаться в мини-прод.

## Базовая локалка (обязательная)
TODO:
- [ ] Postgres single
- [ ] Redis single
- [ ] Kafka single

## Расширенная локалка (опциональная)
TODO:
- [ ] Kafka 3 brokers (для проверки replication) — только если нужно
- [ ] Redis Sentinel/Cluster — только если нужно
- [ ] Postgres HA (Patroni) — обычно не надо локально

## Observability profile
TODO:
- [ ] Prometheus + Grafana
- [ ] Loki или ELK (выбери один; Loki проще)

## Storage profile
TODO:
- [ ] MinIO

## Проверки
TODO:
- [ ] команды smoke-check:
  - postgres ready
  - redis ready
  - kafka produce/consume
