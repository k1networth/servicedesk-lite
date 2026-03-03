variable "name_prefix" {
  description = "Prefix for resource names."
  type        = string
}

variable "folder_id" {
  description = "Yandex Cloud folder ID."
  type        = string
}

variable "network_id" {
  description = "VPC network ID."
  type        = string
}

variable "subnet_id_a" {
  description = "Subnet ID in zone ru-central1-a (primary host)."
  type        = string
}

variable "subnet_id_b" {
  description = "Subnet ID in zone ru-central1-b (replica host)."
  type        = string
}

variable "pg_version" {
  description = "PostgreSQL major version."
  type        = string
  default     = "16"
}

variable "pg_database" {
  description = "Initial database name."
  type        = string
  default     = "servicedesk"
}

variable "pg_user" {
  description = "Initial database user."
  type        = string
  default     = "servicedesk"
}

variable "pg_password" {
  description = "Password for the initial database user."
  type        = string
  sensitive   = true
}

variable "pg_cores" {
  description = "vCPUs per host."
  type        = number
  default     = 2
}

variable "pg_memory" {
  description = "RAM per host in GB."
  type        = number
  default     = 8
}

variable "pg_disk_size" {
  description = "Disk size per host in GB."
  type        = number
  default     = 20
}

variable "backup_retain_period_days" {
  description = "Number of days to retain automated backups."
  type        = number
  default     = 7
}
