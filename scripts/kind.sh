#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-servicedesk}"
CONFIG="${CONFIG:-infra/k8s/kind/kind-config.yaml}"

cmd="${1:-}"

case "$cmd" in
  up)
    kind create cluster --name "$CLUSTER_NAME" --config "$CONFIG"
    ;;
  down)
    kind delete cluster --name "$CLUSTER_NAME"
    ;;
  *)
    echo "Usage: $0 {up|down}"
    echo "  env: CLUSTER_NAME=... CONFIG=..."
    exit 2
    ;;
esac
