# Локальная разработка

## 1) Окружение

Скопируй `.env.example` → `.env`.

**Важно:** в `DATABASE_URL` укажи `127.0.0.1` (а не `localhost`), чтобы избежать резолва `localhost -> ::1`:

```
DATABASE_URL=postgres://servicedesk:servicedesk@127.0.0.1:5432/servicedesk?sslmode=disable
KAFKA_BROKERS=localhost:29092
KAFKA_TOPIC=tickets.events
```

## 2) Поднять инфраструктуру (Postgres + Kafka)

```
docker compose --env-file .env -f infra/local/compose.yaml up -d
```

Проверка:
- Postgres: `localhost:5432`
- Kafka host listener: `localhost:29092`

## 3) Миграции

```
make migrate-up
```

## 4) Запуск сервисов

### Вариант A: E2E-скрипт (рекомендуется)

```
./scripts/e2e_local.sh
```

### Вариант B: вручную (3 терминала)

Терминал 1:
```
make run-ticket
```

Терминал 2:
```
make run-relay
```

Терминал 3:
```
make run-notify
```

## 5) Проверки

Создать тикет:
```
curl -s -X POST http://localhost:8080/tickets   -H 'Content-Type: application/json'   -H "X-Request-Id: rid-$(date +%s)"   -d '{"title":"Diag ticket","description":"check notify"}'
```

Проверить таблицы:
```
./scripts/diag.sh outbox
./scripts/diag.sh processed
```

Посмотреть Kafka:
```
./scripts/diag.sh peek tickets.events
```

## Troubleshooting

- Ошибка миграций вида `[::1]:5432`:
  - замени `localhost` на `127.0.0.1` в `DATABASE_URL`
- Не видишь сообщений в Kafka:
  - внутри контейнера используй `kafka:9092`
  - с хоста используй `localhost:29092`
- Consumer не обрабатывает события:
  - убедись, что `notification-service` запущен
  - проверь лаг: `./scripts/diag.sh group notification-service`
