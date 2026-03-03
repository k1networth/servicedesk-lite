output "network_id" {
  description = "VPC network ID."
  value       = yandex_vpc_network.main.id
}

output "subnet_ids" {
  description = "Map of subnet IDs keyed by zone suffix (a, b, c)."
  value       = { for k, s in yandex_vpc_subnet.subnets : k => s.id }
}

output "security_group_id" {
  description = "Security group ID for the cluster."
  value       = yandex_vpc_security_group.cluster.id
}
