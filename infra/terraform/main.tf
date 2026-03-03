terraform {
  required_version = ">= 1.6"

  required_providers {
    yandex = {
      source  = "yandex-cloud/yandex"
      version = "~> 0.130"
    }
  }

  # Local backend: state is stored in terraform.tfstate next to this file.
  # For production use an S3-compatible remote backend (see backend.hcl.example).
  backend "local" {
    path = "terraform.tfstate"
  }
}

provider "yandex" {
  token     = var.yc_token
  cloud_id  = var.yc_cloud_id
  folder_id = var.yc_folder_id
  zone      = "ru-central1-a"
}

# ── Modules ──────────────────────────────────────────────────────────────────

module "network" {
  source = "./modules/network"

  name_prefix = var.name_prefix
  folder_id   = var.yc_folder_id
}

module "registry" {
  source = "./modules/registry"

  name_prefix       = var.name_prefix
  folder_id         = var.yc_folder_id
  k8s_node_sa_id    = module.k8s.node_sa_id
}

module "k8s" {
  source = "./modules/k8s"

  name_prefix       = var.name_prefix
  folder_id         = var.yc_folder_id
  network_id        = module.network.network_id
  subnet_id_a       = module.network.subnet_ids["a"]
  subnet_ids        = module.network.subnet_ids
  k8s_version       = var.k8s_version
  node_cores        = var.node_cores
  node_memory       = var.node_memory
  node_disk_size    = var.node_disk_size
  node_min          = var.node_min
  node_max          = var.node_max
}

module "postgres" {
  source = "./modules/postgres"

  name_prefix      = var.name_prefix
  folder_id        = var.yc_folder_id
  network_id       = module.network.network_id
  subnet_id_a      = module.network.subnet_ids["a"]
  subnet_id_b      = module.network.subnet_ids["b"]
  pg_version       = var.pg_version
  pg_database      = var.pg_database
  pg_user          = var.pg_user
  pg_password      = var.pg_password
  pg_cores         = var.pg_cores
  pg_memory        = var.pg_memory
  pg_disk_size     = var.pg_disk_size
}
