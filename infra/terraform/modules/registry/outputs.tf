output "registry_id" {
  description = "Container Registry ID."
  value       = yandex_container_registry.main.id
}

output "registry_endpoint" {
  description = "Full registry endpoint for docker push/pull commands."
  value       = "cr.yandex/${yandex_container_registry.main.id}"
}
