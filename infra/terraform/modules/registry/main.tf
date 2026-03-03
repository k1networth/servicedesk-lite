terraform {
  required_providers {
    yandex = {
      source  = "yandex-cloud/yandex"
      version = "~> 0.130"
    }
  }
}

resource "yandex_container_registry" "main" {
  name      = "${var.name_prefix}-registry"
  folder_id = var.folder_id
}

# Allow k8s worker nodes to pull images from this registry
resource "yandex_container_registry_iam_binding" "puller" {
  registry_id = yandex_container_registry.main.id
  role        = "container-registry.images.puller"

  members = [
    "serviceAccount:${var.k8s_node_sa_id}",
  ]
}
