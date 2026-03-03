output "cluster_id" {
  description = "Managed Kubernetes cluster ID."
  value       = module.k8s.cluster_id
}

output "cluster_endpoint" {
  description = "Managed Kubernetes API endpoint (external)."
  value       = module.k8s.cluster_endpoint
}

output "registry_id" {
  description = "Yandex Container Registry ID."
  value       = module.registry.registry_id
}

output "registry_endpoint" {
  description = "Container Registry endpoint for docker push/pull."
  value       = "cr.yandex/${module.registry.registry_id}"
}

output "postgres_host" {
  description = "Managed PostgreSQL cluster FQDN (primary host)."
  value       = module.postgres.host_fqdn
}

output "postgres_port" {
  description = "Managed PostgreSQL port."
  value       = 6432  # pgBouncer port
}

output "postgres_conn_string" {
  description = "PostgreSQL connection string template (password not included)."
  value       = "postgres://${var.pg_user}@${module.postgres.host_fqdn}:6432/${var.pg_database}?sslmode=require"
  sensitive   = false
}

output "network_id" {
  description = "VPC network ID."
  value       = module.network.network_id
}

# Print after apply: helm upgrade command hint
output "helm_upgrade_hint" {
  description = "Helm upgrade command to deploy servicedesk-lite to the created cluster."
  value       = <<-EOT
    # 1. Get kubeconfig:
    yc managed-kubernetes cluster get-credentials ${module.k8s.cluster_id} --external

    # 2. Deploy:
    helm upgrade --install servicedesk-lite infra/k8s/helm/servicedesk-lite \
      -f infra/k8s/helm/servicedesk-lite/values-prod.yaml \
      --set postgres.password=$POSTGRES_PASSWORD \
      --set images.ticketService.repository=cr.yandex/${module.registry.registry_id}/servicedesk/ticket-service \
      --set images.outboxRelay.repository=cr.yandex/${module.registry.registry_id}/servicedesk/outbox-relay \
      --set images.notificationService.repository=cr.yandex/${module.registry.registry_id}/servicedesk/notification-service \
      --set images.migrate.repository=cr.yandex/${module.registry.registry_id}/servicedesk/migrate
  EOT
}
