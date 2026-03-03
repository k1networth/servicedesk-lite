variable "name_prefix" {
  description = "Prefix for resource names."
  type        = string
}

variable "folder_id" {
  description = "Yandex Cloud folder ID."
  type        = string
}

variable "subnets" {
  description = "Map of subnet configs keyed by zone suffix (a, b, c)."
  type = map(object({
    zone = string
    cidr = string
  }))
  default = {
    a = { zone = "ru-central1-a", cidr = "10.10.1.0/24" }
    b = { zone = "ru-central1-b", cidr = "10.10.2.0/24" }
    c = { zone = "ru-central1-c", cidr = "10.10.3.0/24" }
  }
}
