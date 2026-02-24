# TODO / Backlog (по итерациям)

Этот файл — единственное место, где допускаются TODO. В остальных документах должны быть либо готовые инструкции, либо секции "План/Следующая итерация".

## Iteration 3 — Kubernetes deploy
- [ ] Helm chart или Kustomize (выбрать один подход и зафиксировать)
- [ ] Deployments/Services для ticket-service, outbox-relay, notification-service
- [ ] ConfigMap/Secret для DB/Kafka
- [ ] readiness/liveness probes
- [ ] requests/limits
- [ ] Ingress для ticket-service
- [ ] HPA (минимум для ticket-service)
- [ ] PDB + topology spread/anti-affinity (минимально)
- [ ] Скрипт: kind/minikube → одна команда → сервис доступен

## Iteration 4 — Observability
- [ ] Prometheus scrape всех /metrics
- [ ] Grafana dashboard (RPS/latency/errors, outbox dead/failed, notify errors/processed)
- [ ] (опц) Loki + promtail
- [ ] (stretch) OpenTelemetry traces

## Iteration 5 — CI/CD
- [ ] CI: make check + build images
- [ ] Security: govulncheck (+ опц trivy/grype)
- [ ] CD минимум: deploy в kind в CI
- [ ] (опц) GitOps (ArgoCD/Flux)

## Iteration 6 — Дипломные артефакты
- [ ] C4: Context/Container/Component
- [ ] Sequence diagram: create ticket → outbox → relay → kafka → notify
- [ ] Таблица NFR (надёжность, идемпотентность, масштабирование, наблюдаемость)
- [ ] Инструкция воспроизведения демо: local compose + k8s + CI
