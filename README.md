# git i — TODO-only scaffold (high-RPS + HA oriented)

Здесь **нет реализации**: ни Go-кода, ни рабочих Dockerfile/compose/Makefile. Везде **только очень подробные TODO**,
но TODO написаны так, чтобы проект можно было довести до:
- **высокого RPS** (статлес сервисы, правильные индексы, кэш, очереди)
- **HA** (репликации/кластера: Postgres, Kafka, Redis, k8s best practices)
- **autoscaling** (HPA + метрики + лаг Kafka + PDB/affinity)

Цель: простой по домену ServiceDesk (тикеты), но “дотошный” по инженерии.

## Концепт (остается простым)
- Ticket CRUD (create/get/list/close)
- Attachments (S3/MinIO)
- Events через Kafka + Transactional Outbox
- Notifications consumer (идемпотентно) + DLQ

## Ключевые документы
- `docs/00-bootstrap.md` — как стартовать репо с нуля
- `docs/05-nfr.md` — нефункциональные требования (SLO, RPS, HA, scaling)
- `docs/10-contracts.md` — API/Kafka/DB/Redis/metrics спецификация
- `docs/30-local-dev.md` — локальная среда (compose) — **только план**
- `docs/40-docker.md` — Dockerfile’ы — **только план**
- `docs/50-testing.md` — тестирование (unit/integration/contract/load/chaos) — **только план**
- `docs/60-observability.md` — метрики/логи/трейсы/алерты — **только план**
- `docs/70-k8s.md` — k8s: HPA, PDB, anti-affinity, stateful workloads — **только план**
- `docs/80-iac.md`urban-spoon — Terraform/Ansible: структура модулей и HA окружений — **только план**
- `docs/90-capacity.md` — capacity planning и тюнинг (pg, kafka, redis)

## Файлы-заглушки
- `Makefile`, `infra/local/docker-compose.yml`, `build/docker/*.Dockerfile`, `.golangci.yml` — **только TODO** (комментарии)
