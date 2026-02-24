# Terraform + Ansible (план)

Для диплома IaC можно сделать в минимальном виде или описать как "production blueprint".

## Terraform
Планируемая структура:
- modules: network, k8s cluster (managed/self-managed), registry (опц), observability (опц)
- env: dev (обязательно), stage (опц)

## Ansible
Планируемые роли:
- base hardening (sysctl/limits)
- container runtime
- k8s node bootstrap (если self-managed)
- monitoring agents (опц)

## HA design points (что описать)
- multi-AZ node groups
- spread across nodes/az
- backups (Postgres) + retention
- DR notes (опц)

Подробный чек-лист задач — docs/TODO.md.
