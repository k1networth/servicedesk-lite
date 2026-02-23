#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

COMPOSE_FILE="${COMPOSE_FILE:-infra/local/compose.yaml}"
ENV_FILE="${ENV_FILE:-.env}"

PG_CONTAINER="${PG_CONTAINER:-servicedesk-postgres}"
KAFKA_CONTAINER="${KAFKA_CONTAINER:-servicedesk-kafka}"

# Inside kafka container use kafka:9092 (NOT localhost:9092)
KAFKA_BOOTSTRAP_IN_CONTAINER="${KAFKA_BOOTSTRAP_IN_CONTAINER:-kafka:9092}"

TICKET_ADDR="${TICKET_ADDR:-http://localhost:8080}"
METRICS_ADDR_RELAY="${METRICS_ADDR_RELAY:-http://localhost:9090/metrics}"
METRICS_ADDR_NOTIFY="${METRICS_ADDR_NOTIFY:-http://localhost:9091/metrics}"

KAFKA_PEEK_TIMEOUT_MS="${KAFKA_PEEK_TIMEOUT_MS:-5000}"

require() { command -v "$1" >/dev/null 2>&1 || { echo "missing dependency: $1"; exit 2; }; }
require docker
require curl
require python3
require make

has_nc=0
command -v nc >/dev/null 2>&1 && has_nc=1

wait_port() {
  local host="$1" port="$2" name="$3" i
  echo "wait: $name ($host:$port)"
  for i in {1..60}; do
    if [[ "$has_nc" -eq 1 ]]; then
      nc -z "$host" "$port" >/dev/null 2>&1 && return 0
    else
      (echo >/dev/tcp/"$host"/"$port") >/dev/null 2>&1 && return 0
    fi
    sleep 0.5
  done
  echo "timeout waiting for $name ($host:$port)"
  exit 1
}

wait_postgres_ready() {
  echo "wait: postgres ready (pg_isready + select 1)"
  for i in {1..60}; do
    if docker exec -i "$PG_CONTAINER" pg_isready -U "${POSTGRES_USER:-servicedesk}" -d "${POSTGRES_DB:-servicedesk}" >/dev/null 2>&1; then
      docker exec -i "$PG_CONTAINER" psql -U "${POSTGRES_USER:-servicedesk}" -d "${POSTGRES_DB:-servicedesk}" -c "select 1" >/dev/null 2>&1 && return 0
    fi
    sleep 0.5
  done
  echo "timeout waiting postgres ready"
  exit 1
}

json_get() {
  local json="$1" field="$2"
  if command -v jq >/dev/null 2>&1; then
    echo "$json" | jq -r --arg f "$field" '.[$f] // empty'
    return 0
  fi
  python3 -c 'import json,sys; d=json.loads(sys.argv[1]); print(d.get(sys.argv[2], ""))' "$json" "$field"
}

validate_env() {
  # warn about localhost -> ::1 issues
  if [[ "${DATABASE_URL:-}" == *"@localhost:"* ]]; then
    echo "WARN: DATABASE_URL contains localhost. Prefer 127.0.0.1 to avoid ::1 issues."
  fi

  # hard fail on non-ascii topic (e.g. кириллическая 'с')
  local topic="${KAFKA_TOPIC:-tickets.events}"
  if ! python3 -c 'import sys; s=sys.argv[1]; sys.exit(0 if all(ord(c)<128 for c in s) else 1)' "$topic"; then
    echo "ERROR: KAFKA_TOPIC contains non-ASCII characters: '$topic'"
    echo "Fix it to: KAFKA_TOPIC=tickets.events"
    exit 2
  fi
}

cleanup() {
  echo "cleanup: stopping services..."
  [[ -n "${PID_TICKET:-}" ]] && kill "$PID_TICKET" >/dev/null 2>&1 || true
  [[ -n "${PID_RELAY:-}" ]] && kill "$PID_RELAY" >/dev/null 2>&1 || true
  [[ -n "${PID_NOTIFY:-}" ]] && kill "$PID_NOTIFY" >/dev/null 2>&1 || true
}
trap cleanup EXIT

if [[ ! -f "$ENV_FILE" ]]; then
  echo ".env not found -> copying from .env.example"
  cp .env.example .env
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

validate_env

echo "=== compose up ==="
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d

wait_port "127.0.0.1" "5432" "postgres"
wait_postgres_ready

if docker ps --format '{{.Names}}' | grep -q "$KAFKA_CONTAINER"; then
  wait_port "127.0.0.1" "29092" "kafka(host listener)"
fi

echo "=== migrate ==="
for i in {1..10}; do
  if make migrate-up; then
    break
  fi
  echo "migrate failed, retry $i/10"
  sleep 1
done

echo "=== start ticket-service ==="
( set -a; source "$ENV_FILE"; set +a; go run ./cmd/ticket-service ) > /tmp/ticket-service.log 2>&1 &
PID_TICKET=$!

echo "=== start outbox-relay ==="
( set -a; source "$ENV_FILE"; set +a; go run ./cmd/outbox-relay ) > /tmp/outbox-relay.log 2>&1 &
PID_RELAY=$!

echo "=== start notification-service ==="
( set -a; source "$ENV_FILE"; set +a; go run ./cmd/notification-service ) > /tmp/notification-service.log 2>&1 &
PID_NOTIFY=$!

echo "wait: ticket-service healthz"
for i in {1..60}; do
  if curl -fsS "$TICKET_ADDR/healthz" >/dev/null 2>&1; then break; fi
  sleep 0.5
done

echo "=== create ticket ==="
REQ_ID="rid-e2e-$(date +%s)"
RESP="$(curl -fsS -X POST "$TICKET_ADDR/tickets" \
  -H 'Content-Type: application/json' \
  -H "X-Request-Id: $REQ_ID" \
  -d '{"title":"E2E ticket","description":"created by scripts/e2e_local.sh"}')"

TICKET_ID="$(json_get "$RESP" "id")"
if [[ -z "$TICKET_ID" ]]; then
  echo "failed to parse ticket id from response: $RESP"
  echo "ticket-service log: /tmp/ticket-service.log"
  exit 1
fi
echo "ticket_id=$TICKET_ID"

echo "=== ensure topic exists ==="
TOPIC="${KAFKA_TOPIC:-tickets.events}"
docker exec -i "$KAFKA_CONTAINER" bash -lc \
  "/opt/kafka/bin/kafka-topics.sh --bootstrap-server $KAFKA_BOOTSTRAP_IN_CONTAINER \
   --create --if-not-exists --topic $TOPIC --partitions 1 --replication-factor 1" >/dev/null 2>&1 || true

echo "=== wait outbox sent ==="
EVENT_ID=""
for i in {1..80}; do
  row="$(docker exec -i "$PG_CONTAINER" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc \
    "select event_id||'|'||status from outbox where aggregate_id='$TICKET_ID' order by created_at desc limit 1;" 2>/dev/null || true)"
  if [[ -n "$row" ]]; then
    EVENT_ID="${row%%|*}"
    STATUS="${row##*|}"
    if [[ "$STATUS" == "sent" ]]; then
      echo "outbox: event_id=$EVENT_ID status=sent"
      break
    fi
    echo "outbox: event_id=$EVENT_ID status=$STATUS (waiting)"
  else
    echo "outbox: no row yet (waiting)"
  fi
  sleep 0.5
done

if [[ -z "$EVENT_ID" ]]; then
  echo "FAIL: outbox event not sent for ticket_id=$TICKET_ID"
  echo "relay log: /tmp/outbox-relay.log"
  exit 1
fi

echo "=== peek kafka for event_id (best effort) ==="
kafka_out="$(docker exec -i "$KAFKA_CONTAINER" bash -lc \
  "/opt/kafka/bin/kafka-console-consumer.sh \
   --bootstrap-server $KAFKA_BOOTSTRAP_IN_CONTAINER \
   --topic $TOPIC --from-beginning --max-messages 50 --timeout-ms $KAFKA_PEEK_TIMEOUT_MS" 2>/dev/null || true)"

if echo "$kafka_out" | grep -q "$EVENT_ID"; then
  echo "kafka: found event_id=$EVENT_ID"
else
  echo "kafka: could not confirm event_id in peek (ok if timeout)"
fi

echo "=== wait processed_events done ==="
final=""
for i in {1..80}; do
  final="$(docker exec -i "$PG_CONTAINER" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc \
    "select status from processed_events where event_id='$EVENT_ID' limit 1;" 2>/dev/null || true)"
  if [[ "$final" == "done" ]]; then
    echo "processed_events: event_id=$EVENT_ID status=done"
    break
  fi
  sleep 0.5
done

if [[ "$final" != "done" ]]; then
  echo "FAIL: processed_events not done for event_id=$EVENT_ID"
  echo "logs:"
  echo "  /tmp/ticket-service.log"
  echo "  /tmp/outbox-relay.log"
  echo "  /tmp/notification-service.log"
  exit 1
fi

echo "=== metrics sanity ==="
curl -fsS "$METRICS_ADDR_RELAY" >/dev/null 2>&1 && echo "relay metrics OK" || echo "relay metrics not reachable"
curl -fsS "$METRICS_ADDR_NOTIFY" >/dev/null 2>&1 && echo "notify metrics OK" || echo "notify metrics not reachable"

echo "✅ E2E OK: ticket_id=$TICKET_ID event_id=$EVENT_ID"