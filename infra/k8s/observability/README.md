# Observability (kind)

Files:
- kube-prometheus-stack.values.yaml — values for installing Prometheus+Grafana in kind
- servicemonitors.yaml — ServiceMonitor objects for ticket-service/outbox-relay/notification-service
- dashboards/servicedesk-lite.json — minimal Grafana dashboard

Usage:
  make k8s-obs-install
  make k8s-obs-apply
  make k8s-obs-grafana
