terraform {
  required_providers {
    yandex = {
      source  = "yandex-cloud/yandex"
      version = "~> 0.130"
    }
  }
}

resource "yandex_vpc_network" "main" {
  name      = "${var.name_prefix}-vpc"
  folder_id = var.folder_id
}

resource "yandex_vpc_subnet" "subnets" {
  for_each = var.subnets

  name           = "${var.name_prefix}-subnet-${each.key}"
  folder_id      = var.folder_id
  zone           = each.value.zone
  network_id     = yandex_vpc_network.main.id
  v4_cidr_blocks = [each.value.cidr]
}

# NAT gateway — outbound internet access for cluster nodes
resource "yandex_vpc_gateway" "nat" {
  name      = "${var.name_prefix}-nat-gw"
  folder_id = var.folder_id

  shared_egress_gateway {}
}

resource "yandex_vpc_route_table" "nat" {
  name       = "${var.name_prefix}-nat-rt"
  folder_id  = var.folder_id
  network_id = yandex_vpc_network.main.id

  static_route {
    destination_prefix = "0.0.0.0/0"
    gateway_id         = yandex_vpc_gateway.nat.id
  }
}

# Re-create subnets with the NAT route table attached.
# We use a separate set of subnets for the k8s node group that needs outbound
# access (pulling images, etc.).  The same subnets are also used for postgres,
# so we attach the route table to all of them for simplicity.
resource "yandex_vpc_subnet" "subnets_routed" {
  for_each = var.subnets

  name           = "${var.name_prefix}-subnet-routed-${each.key}"
  folder_id      = var.folder_id
  zone           = each.value.zone
  network_id     = yandex_vpc_network.main.id
  v4_cidr_blocks = [cidrsubnet(each.value.cidr, 1, 1)]
  route_table_id = yandex_vpc_route_table.nat.id
}

# Security group: allow internal cluster traffic + postgres access from k8s
resource "yandex_vpc_security_group" "cluster" {
  name       = "${var.name_prefix}-sg-cluster"
  folder_id  = var.folder_id
  network_id = yandex_vpc_network.main.id

  # Allow all traffic within the VPC
  ingress {
    protocol       = "ANY"
    description    = "Internal VPC traffic"
    v4_cidr_blocks = ["10.10.0.0/16"]
  }

  # Allow HTTPS from the internet to the Ingress controller
  ingress {
    protocol       = "TCP"
    description    = "HTTPS from internet"
    v4_cidr_blocks = ["0.0.0.0/0"]
    port           = 443
  }

  # Allow HTTP from the internet (redirect to HTTPS)
  ingress {
    protocol       = "TCP"
    description    = "HTTP from internet"
    v4_cidr_blocks = ["0.0.0.0/0"]
    port           = 80
  }

  # Allow K8s API access
  ingress {
    protocol       = "TCP"
    description    = "Kubernetes API"
    v4_cidr_blocks = ["0.0.0.0/0"]
    port           = 6443
  }

  egress {
    protocol       = "ANY"
    description    = "All outbound traffic"
    v4_cidr_blocks = ["0.0.0.0/0"]
  }
}
