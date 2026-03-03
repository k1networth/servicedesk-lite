terraform {
  required_providers {
    yandex = {
      source  = "yandex-cloud/yandex"
      version = "~> 0.130"
    }
  }
}

# ── Service accounts ─────────────────────────────────────────────────────────

# SA used by the cluster to manage cloud resources (LB, disks, etc.)
resource "yandex_iam_service_account" "cluster" {
  name        = "${var.name_prefix}-k8s-cluster-sa"
  folder_id   = var.folder_id
  description = "Service account for Managed K8s cluster control plane."
}

resource "yandex_resourcemanager_folder_iam_member" "cluster_editor" {
  folder_id = var.folder_id
  role      = "editor"
  member    = "serviceAccount:${yandex_iam_service_account.cluster.id}"
}

# SA used by worker nodes (pull images from Container Registry, etc.)
resource "yandex_iam_service_account" "node" {
  name        = "${var.name_prefix}-k8s-node-sa"
  folder_id   = var.folder_id
  description = "Service account for Managed K8s worker nodes."
}

resource "yandex_resourcemanager_folder_iam_member" "node_cr_puller" {
  folder_id = var.folder_id
  role      = "container-registry.images.puller"
  member    = "serviceAccount:${yandex_iam_service_account.node.id}"
}

# ── Managed Kubernetes cluster ───────────────────────────────────────────────

resource "yandex_kubernetes_cluster" "main" {
  name        = "${var.name_prefix}-k8s"
  folder_id   = var.folder_id
  description = "Managed Kubernetes cluster for servicedesk-lite."

  network_id = var.network_id

  master {
    version = var.k8s_version

    # Regional master spans three zones — high availability control plane.
    regional {
      region = "ru-central1"

      dynamic "location" {
        for_each = var.subnet_ids
        content {
          zone      = "ru-central1-${location.key}"
          subnet_id = location.value
        }
      }
    }

    public_ip = true  # expose K8s API externally so CI/CD can deploy

    maintenance_policy {
      auto_upgrade = true

      maintenance_window {
        day        = "sunday"
        start_time = "02:00"
        duration   = "3h"
      }
    }
  }

  service_account_id      = yandex_iam_service_account.cluster.id
  node_service_account_id = yandex_iam_service_account.node.id

  release_channel = "STABLE"

  depends_on = [
    yandex_resourcemanager_folder_iam_member.cluster_editor,
    yandex_resourcemanager_folder_iam_member.node_cr_puller,
  ]
}

# ── Node group (auto-scaling) ─────────────────────────────────────────────────

resource "yandex_kubernetes_node_group" "main" {
  cluster_id  = yandex_kubernetes_cluster.main.id
  name        = "${var.name_prefix}-ng"
  description = "Auto-scaling worker node group."
  version     = var.k8s_version

  instance_template {
    platform_id = "standard-v3"  # Intel Ice Lake

    resources {
      cores         = var.node_cores
      memory        = var.node_memory
      core_fraction = 100
    }

    boot_disk {
      type = "network-ssd"
      size = var.node_disk_size
    }

    network_interface {
      subnet_ids = values(var.subnet_ids)
      nat        = false  # nodes use NAT gateway, not public IPs
    }

    container_runtime {
      type = "containerd"
    }
  }

  scale_policy {
    auto_scale {
      min     = var.node_min
      max     = var.node_max
      initial = var.node_min
    }
  }

  allocation_policy {
    dynamic "location" {
      for_each = var.subnet_ids
      content {
        zone = "ru-central1-${location.key}"
      }
    }
  }

  maintenance_policy {
    auto_upgrade = true
    auto_repair  = true

    maintenance_window {
      day        = "sunday"
      start_time = "03:00"
      duration   = "2h"
    }
  }
}
