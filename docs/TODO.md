# TODO / Backlog (по итерациям)

Этот файл — единственное место, где допускаются TODO. В остальных документах должны быть либо готовые инструкции, либо секции "План/Следующая итерация".

## Iteration 3 — Kubernetes deploy ✅ Выполнено

- [x] Helm chart (выбран, реализован)
- [x] Deployments/Services для ticket-service, outbox-relay, notification-service
- [x] ConfigMap/Secret для DB/Kafka
- [x] readiness/liveness probes (с реальным pg.PingContext)
- [x] requests/limits для всех сервисов
- [x] Ingress для ticket-service (и Grafana/Prometheus)
- [x] HPA для ticket-service (min 1, max 3, CPU 70%)
- [x] PDB (minAvailable: 1)
- [x] `make k8s-up` — одна команда → сервис доступен

## Iteration 4 — Observability ✅ Выполнено

- [x] Prometheus scrape всех /metrics (ServiceMonitors)
- [x] Grafana dashboard (RPS, outbox lag, notify processed)
- [ ] (опц) Loki + promtail
- [ ] (stretch) OpenTelemetry traces

## Iteration 5 — CI/CD

- [x] CI: lint + test + build + images (GitHub Actions, `.github/workflows/ci.yaml`)
- [x] Security: govulncheck
- [ ] CD минимум: deploy в kind в CI (smoke test)
- [ ] (опц) GitOps (ArgoCD/Flux)

## Iteration 6 — Дипломные артефакты ✅ Выполнено

- [x] C4: Context/Container диаграммы (Mermaid, `docs/95-diagrams.md`)
- [x] Sequence diagram: create ticket → outbox → relay → kafka → notify
- [x] Sequence diagram: CI/CD пайплайн
- [x] Инструкция воспроизведения демо: local compose + k8s + CI (`docs/99-demo.md`)
