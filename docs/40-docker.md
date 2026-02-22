# Docker — TODO (prod-friendly images)

## Общие требования
TODO:
- [ ] multi-stage
- [ ] non-root runtime
- [ ] минимальный runtime
- [ ] сигналы SIGTERM/SIGINT корректно обрабатываются приложением
- [ ] конфиг через env

## ticket-service
TODO:
- [ ] EXPOSE 8080
- [ ] healthcheck -> /healthz (опционально)

## relay/consumer
TODO:
- [ ] без портов
- [ ] корректный shutdown: stop polling/stop consumer, commit offsets where needed
