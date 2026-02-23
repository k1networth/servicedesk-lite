#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

ENV_FILE="${ENV_FILE:-.env}"
COMPOSE_FILE="${COMPOSE_FILE:-infra/local/compose.yaml}"

PG_CONTAINER="${PG_CONTAINER:-servicedesk-postgres}"
KAFKA_CONTAINER="${KAFKA_CONTAINER:-servicedesk-kafka}"
KAFKA_BOOTSTRAP_IN_CONTAINER="${KAFKA_BOOTSTRAP_IN_CONTAINER:-kafka:9092}"
KAFKA_PEEK_TIMEOUT_MS="${KAFKA_PEEK_TIMEOUT_MS:-5000}"

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
fi

cmd="${1:-help}"

case "$cmd" in
  up)
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d
    ;;
  down)
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" down -v
    ;;
  topics)
    docker exec -it "$KAFKA_CONTAINER" bash -lc \
      "/opt/kafka/bin/kafka-topics.sh --bootstrap-server $KAFKA_BOOTSTRAP_IN_CONTAINER --list"
    ;;
  peek)
    topic="${2:-${KAFKA_TOPIC:-tickets.events}}"
    docker exec -it "$KAFKA_CONTAINER" bash -lc \
      "/opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server $KAFKA_BOOTSTRAP_IN_CONTAINER \
       --topic $topic --from-beginning --max-messages 20 --timeout-ms $KAFKA_PEEK_TIMEOUT_MS"
    ;;
  groups)
    docker exec -it "$KAFKA_CONTAINER" bash -lc \
      "/opt/kafka/bin/kafka-consumer-groups.sh --bootstrap-server $KAFKA_BOOTSTRAP_IN_CONTAINER --list"
    ;;
  group)
    gid="${2:-${KAFKA_GROUP_ID:-notification-service}}"
    docker exec -it "$KAFKA_CONTAINER" bash -lc \
      "/opt/kafka/bin/kafka-consumer-groups.sh --bootstrap-server $KAFKA_BOOTSTRAP_IN_CONTAINER --describe --group $gid || true"
    ;;
  outbox)
    docker exec -it "$PG_CONTAINER" psql -U "${POSTGRES_USER:-servicedesk}" -d "${POSTGRES_DB:-servicedesk}" \
      -c "select id,event_id,status,attempts,created_at,sent_at from outbox order by created_at desc limit 10;"
    ;;
  processed)
    docker exec -it "$PG_CONTAINER" psql -U "${POSTGRES_USER:-servicedesk}" -d "${POSTGRES_DB:-servicedesk}" \
      -c "select event_id,event_type,status,attempts,first_seen_at,processed_at from processed_events order by first_seen_at desc limit 10;"
    ;;
  help|*)
    echo "usage: ./scripts/diag.sh <cmd>"
    echo "cmd:"
    echo "  up | down"
    echo "  topics | peek [topic]"
    echo "  groups | group [group_id]"
    echo "  outbox | processed"
    ;;
esac