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
  description = "Subnet ID in zone ru-central1-a (for the control plane)."
  type        = string
}

variable "subnet_ids" {
  description = "Map of subnet IDs keyed by zone suffix, used for the node group."
  type        = map(string)
}

variable "k8s_version" {
  description = "Kubernetes version."
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
  description = "Minimum number of worker nodes."
  type        = number
  default     = 2
}

variable "node_max" {
  description = "Maximum number of worker nodes."
  type        = number
  default     = 6
}
