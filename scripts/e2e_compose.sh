#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

COMPOSE_FILE="${COMPOSE_FILE:-infra/local/compose.core.yaml}"
ENV_FILE="${ENV_FILE:-.env}"

PG_CONTAINER="${PG_CONTAINER:-servicedesk-postgres}"
KAFKA_CONTAINER="${KAFKA_CONTAINER:-servicedesk-kafka}"
KAFKA_BOOTSTRAP_IN_CONTAINER="${KAFKA_BOOTSTRAP_IN_CONTAINER:-kafka:9092}"

TICKET_ADDR="${TICKET_ADDR:-http://localhost:8080}"

# E2E topic/group defaults
# E2E should be deterministic and not depend on caller env/.env file
E2E_TS="$(date +%s)"
export KAFKA_START_OFFSET="${KAFKA_START_OFFSET:-first}"
export KAFKA_GROUP_ID="${KAFKA_GROUP_ID:-notification-service-e2e-${E2E_TS}}"
export KAFKA_TOPIC="${KAFKA_TOPIC:-tickets.events.e2e.${E2E_TS}}"
TOPIC="$KAFKA_TOPIC"

require() { command -v "$1" >/dev/null 2>&1 || { echo "missing dependency: $1"; exit 2; }; }
require docker
require curl
require python3

cleanup() {
  if [[ "${E2E_KEEP_STACK:-0}" == "1" ]]; then
    echo "cleanup: E2E_KEEP_STACK=1 -> keep compose stack running"
    return 0
  fi
  echo "cleanup: docker compose down"
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" down -v >/dev/null 2>&1 || true
}

trap cleanup EXIT

if [[ ! -f "$ENV_FILE" ]]; then
  echo ".env not found -> copying from .env.example"
  cp .env.example "$ENV_FILE"
fi

echo "=== compose up (core) ==="
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --build

echo "=== wait: ticket-service healthz ==="
for _ in {1..120}; do
  if curl -fsS "$TICKET_ADDR/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
done
curl -fsS "$TICKET_ADDR/healthz" >/dev/null 2>&1 || {
  echo "FAIL: ticket-service not healthy"
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" ps || true
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" logs --tail=200 ticket-service || true
  exit 1
}


echo "=== wait: outbox-relay metrics ==="
for _ in {1..120}; do
  if curl -fsS "http://localhost:9090/metrics" >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
done
curl -fsS "http://localhost:9090/metrics" >/dev/null 2>&1 || {
  echo "FAIL: outbox-relay metrics not reachable"
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" logs --tail=200 outbox-relay || true
  exit 1
}

echo "=== wait: notification-service metrics ==="
for _ in {1..120}; do
  if curl -fsS "http://localhost:9091/metrics" >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
done
curl -fsS "http://localhost:9091/metrics" >/dev/null 2>&1 || {
  echo "FAIL: notification-service metrics not reachable"
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" logs --tail=200 notification-service || true
  exit 1
}

echo "=== ensure topic exists ==="
docker exec -i "$KAFKA_CONTAINER" bash -lc \
  "/opt/kafka/bin/kafka-topics.sh --bootstrap-server $KAFKA_BOOTSTRAP_IN_CONTAINER \
   --create --if-not-exists --topic $TOPIC --partitions 1 --replication-factor 1" >/dev/null 2>&1 || true

create_ticket() {
  local title="$1" desc="$2" req_id="$3"
  curl -fsS -X POST "$TICKET_ADDR/tickets" \
    -H 'Content-Type: application/json' \
    -H "X-Request-Id: $req_id" \
    -d "{\"title\":\"$title\",\"description\":\"$desc\"}"
}

json_get() {
  local json="$1" field="$2"
  python3 -c 'import json,sys; d=json.loads(sys.argv[1]); print(d.get(sys.argv[2], ""))' "$json" "$field"
}

wait_outbox_sent() {
  local ticket_id="$1" timeout_iters="${2:-120}"
  local row event_id status
  for _ in $(seq 1 "$timeout_iters"); do
    row="$(docker exec -i "$PG_CONTAINER" psql -U "${POSTGRES_USER:-servicedesk}" -d "${POSTGRES_DB:-servicedesk}" -Atc \
      "select event_id||'|'||status from outbox where aggregate_id='$ticket_id' order by created_at desc limit 1;" 2>/dev/null || true)"
    if [[ -n "$row" ]]; then
      event_id="${row%%|*}"
      status="${row##*|}"
      if [[ "$status" == "sent" ]]; then
        echo "$event_id"
        return 0
      fi
      echo "outbox: event_id=$event_id status=$status (waiting for sent)" >&2
    fi
    sleep 0.5
  done
  return 1
}

wait_processed_done() {
  local event_id="$1" timeout_iters="${2:-120}"
  local status
  for _ in $(seq 1 "$timeout_iters"); do
    status="$(docker exec -i "$PG_CONTAINER" psql -U "${POSTGRES_USER:-servicedesk}" -d "${POSTGRES_DB:-servicedesk}" -Atc \
      "select status from processed_events where event_id='$event_id' limit 1;" 2>/dev/null || true)"
    if [[ "$status" == "done" ]]; then
      return 0
    fi
    sleep 0.5
  done
  return 1
}

echo "=== create ticket ==="
REQ_ID="rid-e2e-compose-$(date +%s)"
RESP="$(create_ticket "E2E compose" "stack from docker compose" "$REQ_ID")"
TICKET_ID="$(json_get "$RESP" "id")"
[[ -n "$TICKET_ID" ]] || { echo "FAIL: can't parse ticket id: $RESP"; exit 1; }
echo "ticket_id=$TICKET_ID"

echo "=== wait outbox sent ==="
EVENT_ID="$(wait_outbox_sent "$TICKET_ID" 120)" || {
  echo "FAIL: outbox not sent for ticket_id=$TICKET_ID"
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" logs --tail=200 outbox-relay || true
  exit 1
}
echo "event_id=$EVENT_ID"

echo "=== wait processed_events done ==="
if ! wait_processed_done "$EVENT_ID" 120; then
  echo "FAIL: processed_events not done for event_id=$EVENT_ID"
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" logs --tail=200 notification-service || true
  docker exec -i "$PG_CONTAINER" psql -U "${POSTGRES_USER:-servicedesk}" -d "${POSTGRES_DB:-servicedesk}" -c "select * from processed_events order by first_seen_at desc limit 5" || true
  exit 1
fi

echo "✅ E2E (compose core) OK: ticket_id=$TICKET_ID event_id=$EVENT_ID"
