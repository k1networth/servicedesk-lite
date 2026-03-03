# Документация (единый вход)

Вся документация проекта хранится **в этой папке**.

## Навигация

### База
- Нефункциональные требования (SLO/RPS/HA/ошибки): `05-nfr.md`
- Контракты (HTTP API / формат ошибок / события / метрики): `10-contracts.md`
- События (Kafka envelope): `40-events.md`

### Разработка
- Локальная разработка: `30-local-dev.md`
- Docker/образы: `40-docker.md`
- Тестирование (unit/integration/load): `50-testing.md`

### Эксплуатация
- Observability (metrics/logs/traces): `60-observability.md`
- Kubernetes (Helm/HPA/PDB/rollout/HA): `70-k8s.md`
- IaC / Облако (Terraform, Yandex Cloud blueprint): `80-iac.md`
- CI/CD (GitHub Actions: lint/test/build/deploy): `85-cicd.md`
- Capacity planning & tuning: `90-capacity.md`

### Дипломные артефакты
- Диаграммы (C4 Context/Container, Sequence): `95-diagrams.md`
- Воспроизведение демо (local / k8s / CI): `99-demo.md`

## Документация по сервисам

- Индекс сервисов: `services/README.md`
