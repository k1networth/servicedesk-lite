variable "name_prefix" {
  description = "Prefix for resource names."
  type        = string
}

variable "folder_id" {
  description = "Yandex Cloud folder ID."
  type        = string
}

variable "k8s_node_sa_id" {
  description = "Service account ID of the K8s worker nodes (granted pull access)."
  type        = string
}
