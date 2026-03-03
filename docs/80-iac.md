# IaC — облачная инфраструктура (Yandex Cloud)

Этот документ описывает production-топологию на базе Yandex Cloud.
Terraform-код находится в `infra/terraform/`, Ansible — в `infra/ansible/`.

> **Статус:** реализовано. В демо-среде (kind) зависимости запущены как StatefulSet внутри кластера. Terraform + Ansible описывают целевую архитектуру для production-деплоя.

---

## Целевая топология

```
Yandex Cloud
├── VPC
│   ├── subnet-a (ru-central1-a)
│   ├── subnet-b (ru-central1-b)
│   └── subnet-c (ru-central1-c)
│
├── Managed Service for Kubernetes
│   ├── Control Plane (managed, multi-master)
│   └── Node Group (2–6 нодов, auto-scaling)
│       ├── ticket-service  (Deployment, HPA)
│       ├── outbox-relay    (Deployment)
│       ├── notification-service (Deployment)
│       └── kube-prometheus-stack (Observability)
│
├── Managed Service for PostgreSQL
│   ├── Primary (ru-central1-a)
│   └── Replica (ru-central1-b)  ← failover
│
├── Container Registry
│   └── servicedesk/* (образы сервисов)
│
└── Application Load Balancer
    └── → Ingress Controller (ingress-nginx в кластере)
```

## Почему Managed-сервисы вместо StatefulSet

| Компонент | StatefulSet (demo) | Managed (production) |
|---|---|---|
| Postgres | ручное управление PVC, backups | автобэкапы, failover, мониторинг |
| Отказоустойчивость | при сбое ноды данные могут быть недоступны | multi-AZ replica, автопереключение |
| Обслуживание | ручные патчи, вакуум | автоматические minor updates |
| Масштабирование | ручное изменение ресурсов пода | вертикальное через Yandex Cloud UI/API |

## Структура Terraform

```
infra/terraform/
├── main.tf               # provider + local backend
├── variables.tf          # все входные параметры
├── outputs.tf            # cluster endpoint, registry ID, postgres FQDN
├── backend.hcl.example   # шаблон для S3 backend (production)
└── modules/
    ├── network/          # VPC, subnets, security groups, NAT gateway
    ├── k8s/              # Managed K8s cluster + node group + IAM SA
    ├── postgres/         # Managed PostgreSQL 16, replica, pgBouncer
    └── registry/         # Container Registry + IAM binding
```

По умолчанию state хранится локально (`terraform.tfstate`).
Для production — переключиться на S3 backend (Yandex Object Storage), пример в `backend.hcl.example`:

```bash
cp infra/terraform/backend.hcl.example infra/terraform/backend.hcl
# заполнить access_key / secret_key
terraform init -backend-config=backend.hcl
```

### Модуль network

- VPC с тремя подсетями в разных зонах доступности
- Security group: разрешает трафик только между компонентами (k8s → postgres, egress)
- NAT Gateway для outbound трафика нодов

### Модуль k8s

- `yandex_kubernetes_cluster` — managed control plane
- `yandex_kubernetes_node_group` — auto-scaling group (min 2, max 6)
- Service Account с ролями для Container Registry и Load Balancer

### Модуль postgres

- `yandex_mdb_postgresql_cluster` — managed Postgres 16
- Replica в другой зоне (ru-central1-b)
- Автобэкапы: retention 7 дней
- Connection pooler (pgBouncer) — встроенный в managed сервис

### Модуль registry

- `yandex_container_registry` — приватный реестр образов
- IAM-роль для pull из k8s нодов

## Деплой приложения в production

После `terraform apply` — деплой через тот же Helm chart:

```bash
# Получить kubeconfig
yc managed-kubernetes cluster get-credentials <cluster-name> --external

# Деплой
helm upgrade --install servicedesk-lite \
  infra/k8s/helm/servicedesk-lite \
  -f infra/k8s/helm/servicedesk-lite/values-prod.yaml \
  --set postgres.password=$POSTGRES_PASSWORD
```

`values-prod.yaml` переопределяет:
- `postgres.image` → не используется (Managed Postgres через DATABASE_URL)
- `kafka.image` → не используется (Managed Kafka через KAFKA_BROKERS)
- `images.*.repository` → Container Registry endpoint

## HA-характеристики production-топологии

| Сценарий | Поведение |
|---|---|
| Падение пода ticket-service | HPA + readinessProbe: трафик уходит на другие реплики |
| Rolling update | PDB + maxUnavailable=0: нулевой даунтайм |
| Падение k8s ноды | Kubernetes переносит поды на другой нод автоматически |
| Падение primary Postgres | Managed Service автоматически переключает на реплику |
| Перегрузка ticket-service | HPA масштабирует до 3 (или больше) реплик по CPU |

## Оценка стоимости (Yandex Cloud, ru-central1)

| Компонент | Конфигурация | Стоимость/мес (ориентир) |
|---|---|---|
| Managed K8s Control Plane | — | ~1 500 руб |
| Node Group (2 × 2 vCPU / 8 GB) | standard-v3 | ~6 000 руб |
| Managed PostgreSQL (2 vCPU / 8 GB) | с репликой | ~5 000 руб |
| Container Registry | 10 GB | ~100 руб |
| Application Load Balancer | — | ~500 руб |
| **Итого** | | **~13 000 руб/мес** |

> Для диплома и краткосрочного теста достаточно минимальной конфигурации (~5 000 руб/мес или ~150 руб/день при почасовой оплате).

## Ansible

Playbook для self-managed кластера — альтернатива Managed K8s (например, bare-metal или VMs без managed-сервиса).
Код находится в `infra/ansible/`.

```
infra/ansible/
├── inventory/
│   └── hosts.yml           # шаблон инвентаря (control-plane + workers)
├── group_vars/
│   └── all.yml             # версии K8s, containerd, CNI; pod/service CIDR
├── roles/
│   ├── base/               # отключение swap, kernel modules, sysctl, ulimits
│   ├── container-runtime/  # containerd + runc + CNI plugins
│   └── k8s-node/           # kubeadm init (control-plane) + join (workers)
└── site.yml                # главный playbook
```

### Запуск

```bash
# Заполнить inventory/hosts.yml реальными IP-адресами нод
ansible-playbook -i infra/ansible/inventory/hosts.yml infra/ansible/site.yml
```

Playbook последовательно применяет три роли на все ноды:
1. `base` — ОС-prerequisites
2. `container-runtime` — containerd
3. `k8s-node` — kubeadm bootstrap: init на control-plane, join на workers

В данном проекте основной путь деплоя — Managed K8s (Yandex Cloud), поэтому Ansible описывает on-premise альтернативу.
