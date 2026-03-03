# ── Yandex Cloud auth ────────────────────────────────────────────────────────

variable "yc_token" {
  description = "Yandex Cloud IAM token or OAuth token. Set via TF_VAR_yc_token env var."
  type        = string
  sensitive   = true
}

variable "yc_cloud_id" {
  description = "Yandex Cloud ID."
  type        = string
}

variable "yc_folder_id" {
  description = "Yandex Cloud folder ID."
  type        = string
}

# ── General ───────────────────────────────────────────────────────────────────

variable "name_prefix" {
  description = "Prefix for all resource names (e.g. 'sdl' for servicedesk-lite)."
  type        = string
  default     = "sdl"
}

# ── Kubernetes ────────────────────────────────────────────────────────────────

variable "k8s_version" {
  description = "Kubernetes version for Managed K8s cluster."
  type        = string
  default     = "1.30"
}

variable "node_cores" {
  description = "Number of vCPUs per worker node."
  type        = number
  default     = 2
}

variable "node_memory" {
  description = "RAM per worker node in GB."
  type        = number
  default     = 8
}

variable "node_disk_size" {
  description = "Boot disk size per worker node in GB."
  type        = number
  default     = 64
}

variable "node_min" {
  description = "Minimum number of worker nodes (auto-scaling)."
  type        = number
  default     = 2
}

variable "node_max" {
  description = "Maximum number of worker nodes (auto-scaling)."
  type        = number
  default     = 6
}

# ── PostgreSQL ────────────────────────────────────────────────────────────────

variable "pg_version" {
  description = "PostgreSQL major version."
  type        = string
  default     = "16"
}

variable "pg_database" {
  description = "PostgreSQL database name."
  type        = string
  default     = "servicedesk"
}

variable "pg_user" {
  description = "PostgreSQL user name."
  type        = string
  default     = "servicedesk"
}

variable "pg_password" {
  description = "PostgreSQL user password. Set via TF_VAR_pg_password env var."
  type        = string
  sensitive   = true
}

variable "pg_cores" {
  description = "vCPUs for managed PostgreSQL host."
  type        = number
  default     = 2
}

variable "pg_memory" {
  description = "RAM for managed PostgreSQL host in GB."
  type        = number
  default     = 8
}

variable "pg_disk_size" {
  description = "Disk size for managed PostgreSQL in GB."
  type        = number
  default     = 20
}
