terraform {
  required_providers {
    yandex = {
      source  = "yandex-cloud/yandex"
      version = "~> 0.130"
    }
  }
}

resource "yandex_mdb_postgresql_cluster" "main" {
  name        = "${var.name_prefix}-pg"
  folder_id   = var.folder_id
  environment = "PRODUCTION"
  network_id  = var.network_id

  config {
    version = var.pg_version

    resources {
      resource_preset_id = "s3-c${var.pg_cores}-m${var.pg_memory}"
      disk_type_id       = "network-ssd"
      disk_size          = var.pg_disk_size
    }

    postgresql_config = {
      max_connections                = 200
      shared_buffers                 = 268435456  # 256MB in bytes
      effective_cache_size           = 2147483648 # 2GB in bytes
      default_statistics_target      = 100
      random_page_cost               = 1.1        # SSD
    }

    # Built-in pgBouncer connection pooler — available for managed postgres.
    pooler_config {
      pool_discard = true
      pooling_mode = "SESSION"
    }

    # Automated backups: daily at 03:00 UTC, retained for 7 days.
    backup_window_start {
      hours   = 3
      minutes = 0
    }

    backup_retain_period_days = var.backup_retain_period_days

    access {
      web_sql    = false
      serverless = false
    }
  }

  # Primary host in ru-central1-a
  host {
    zone      = "ru-central1-a"
    subnet_id = var.subnet_id_a
    assign_public_ip = false
  }

  # Replica host in ru-central1-b — automatic failover
  host {
    zone                    = "ru-central1-b"
    subnet_id               = var.subnet_id_b
    assign_public_ip        = false
    replication_source_name = "${var.name_prefix}-pg-host-a"
  }
}

resource "yandex_mdb_postgresql_database" "main" {
  cluster_id = yandex_mdb_postgresql_cluster.main.id
  name       = var.pg_database
  owner      = yandex_mdb_postgresql_user.main.name
  lc_collate = "en_US.UTF-8"
  lc_type    = "en_US.UTF-8"
}

resource "yandex_mdb_postgresql_user" "main" {
  cluster_id = yandex_mdb_postgresql_cluster.main.id
  name       = var.pg_user
  password   = var.pg_password

  permission {
    database_name = var.pg_database
  }

  conn_limit = 50
}
