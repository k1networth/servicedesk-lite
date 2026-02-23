# Local dev (docker compose)

Локальная среда должна помогать разработке, но не превращаться в мини-прод. Поэтому зависимости добавляем **эволюционно**.

## База (уже есть)

### Postgres (single)

Файл: `infra/local/compose.yaml`

Команды:

```bash
cp .env.example .env
make db-up
make db-ps
make migrate-up
```

Остановить и удалить volume:

```bash
make db-down
```

> Важно: `migrate-up` требует переменную `DATABASE_URL` (см. `.env.example`).

## Базовая локалка (дальше по итерациям)
План:
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
