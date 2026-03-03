output "cluster_id" {
  description = "Managed PostgreSQL cluster ID."
  value       = yandex_mdb_postgresql_cluster.main.id
}

output "host_fqdn" {
  description = "FQDN of the primary PostgreSQL host."
  value       = yandex_mdb_postgresql_cluster.main.host[0].fqdn
}

output "database" {
  description = "Database name."
  value       = yandex_mdb_postgresql_database.main.name
}

output "user" {
  description = "Database user name."
  value       = yandex_mdb_postgresql_user.main.name
}
