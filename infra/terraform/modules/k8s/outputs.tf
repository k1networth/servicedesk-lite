output "cluster_id" {
  description = "Managed Kubernetes cluster ID."
  value       = yandex_kubernetes_cluster.main.id
}

output "cluster_endpoint" {
  description = "Managed Kubernetes API server endpoint."
  value       = yandex_kubernetes_cluster.main.master[0].external_v4_endpoint
}

output "node_sa_id" {
  description = "Service account ID used by worker nodes (passed to registry module for IAM binding)."
  value       = yandex_iam_service_account.node.id
}

output "cluster_ca_certificate" {
  description = "Cluster CA certificate (PEM)."
  value       = yandex_kubernetes_cluster.main.master[0].cluster_ca_certificate
  sensitive   = true
}
