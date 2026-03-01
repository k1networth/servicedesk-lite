# Observability

## Метрики
- ticket-service: `http://localhost:8080/metrics`
- outbox-relay: `http://localhost:9090/metrics`
- notification-service: `http://localhost:9091/metrics`

## K8s demo (Prometheus + Grafana)

Установка (kind + наш Helm chart должны быть уже установлены):

1) Поставить стек наблюдаемости:

```bash
make k8s-obs-install
```

2) Подключить наши сервисы к Prometheus (ServiceMonitor) + залить дашборд:

```bash
make k8s-obs-apply
```

3) Открыть Grafana:

```bash
make k8s-obs-ui
```

Доступ (через Ingress, без port-forward):
- Grafana: `http://localhost:8080/grafana` (логин/пароль: `admin/admin`)
- Prometheus: `http://localhost:8080/prometheus`

Примечание: в kind демо ingress прокинут на `:8080` через `infra/k8s/kind/kind-config.yaml`.

## Рекомендуемые метрики
- ticket-service:
  - requests total (route/status)
  - latency histogram (если добавишь)
- outbox-relay:
  - published total / failed total
  - outbox lag seconds (now - oldest pending)
- notification-service:
  - `notify_processed_total{event_type,status}`

## Корреляция
- Используй заголовок `X-Request-Id` для HTTP.
- `request_id` может прокидываться в Kafka envelope для трассировки.
