# CI/CD (GitHub Actions)

CI/CD — ключевой DevOps-инструмент проекта. Автоматизирует проверку кода, сборку образов и деплой.

Конфигурация: `.github/workflows/`

---

## Пайплайн

```
Push / PR
  └─► ci.yaml
        ├─► lint       (golangci-lint)
        ├─► test       (go test ./...)
        ├─► build      (go build ./...)
        └─► [main] images  ──► Container Registry
                              └─► [main] deploy-kind
```

### Джоб: lint

```yaml
- uses: actions/setup-go@v5
- run: go vet ./...
- uses: golangci-lint-run/golangci-lint-action@v6
```

Проверяет: `errcheck`, `staticcheck`, `govet`, `unused`.

### Джоб: test

```yaml
- run: go test -race -count=1 ./...
```

Запускает юнит-тесты с race detector. Не требует Postgres/Kafka — unit-тесты изолированы.

### Джоб: build

```yaml
- run: go build ./cmd/ticket-service ./cmd/outbox-relay ./cmd/notification-service
```

Проверяет компилируемость всех бинарников.

### Джоб: images (только main)

Собирает Docker-образы и пушит в Container Registry:

```yaml
- uses: docker/build-push-action@v5
  with:
    context: .
    file: cmd/ticket-service/Dockerfile
    push: true
    tags: cr.yandex/<registry-id>/servicedesk/ticket-service:${{ github.sha }}
```

Аналогично для `outbox-relay`, `notification-service`, `migrate`.

### Джоб: deploy-kind (только main, опционально)

Деплой в kind-кластер внутри CI-раннера для smoke-теста:

```bash
kind create cluster
helm upgrade --install servicedesk-lite infra/k8s/helm/servicedesk-lite \
  -f infra/k8s/helm/servicedesk-lite/values-kind.yaml
kubectl rollout status deployment/ticket-service -n servicedesk
curl -f http://localhost:8080/healthz
```

---

## Безопасность

### govulncheck

```yaml
- uses: golang/govulncheck-action@v1
```

Проверяет зависимости на известные CVE из базы Go Vulnerability Database.

### Сканирование образов (опц)

```yaml
- uses: aquasecurity/trivy-action@master
  with:
    image-ref: cr.yandex/.../ticket-service:${{ github.sha }}
    severity: HIGH,CRITICAL
    exit-code: 1
```

---

## Переменные окружения CI

| Секрет | Назначение |
|---|---|
| `YC_REGISTRY_ID` | ID Container Registry в Yandex Cloud |
| `YC_SA_JSON_CREDENTIALS` | Service Account ключ для docker login |

---

## Итоговая схема DevOps-цикла

```
Developer
  └─► git push
        └─► GitHub Actions CI
              ├─► lint + test + build   (все ветки)
              ├─► govulncheck            (все ветки)
              └─► build images + push   (main)
                    └─► deploy to kind  (smoke test)
                          └─► helm upgrade --install
                                └─► kubectl rollout status
```

Полный цикл от пуша до задеплоенного образа — менее 3 минут.
