# Terraform + Ansible — TODO (HA environment)

## Terraform (infra/terraform)
TODO:
- [ ] modules:
  - network
  - k8s cluster (managed/self-managed)
  - registry (optional)
  - observability stack (optional)
- [ ] envs:
  - dev
  - stage (optional)

## Ansible (infra/ansible)
TODO:
- [ ] roles:
  - base hardening (sysctl, limits)
  - container runtime
  - k8s node bootstrap (если self-managed)
  - monitoring agents (optional)

## HA design points to document
TODO:
- [ ] multi-AZ node groups
- [ ] spread across nodes/az
- [ ] backups (Postgres) + retention
- [ ] disaster recovery notes (optional)
