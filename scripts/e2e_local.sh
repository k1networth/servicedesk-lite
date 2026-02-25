#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

COMPOSE_FILE="${COMPOSE_FILE:-infra/local/compose.yaml}"
BASE_ENV_FILE="${ENV_FILE:-.env}"

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

usage() {
  cat <<'EOF'
Usage:
  ./scripts/e2e_local.sh                  # happy-path E2E
  ./scripts/e2e_local.sh --demo           # demo: notify -> DLQ + processed_events=failed AND outbox -> failed
  ./scripts/e2e_local.sh --demo-notify    # only notify DLQ demo
  ./scripts/e2e_local.sh --demo-outbox    # only outbox failed demo (Kafka down)
EOF
}

MODE="success"
case "${1:-}" in
  --demo) MODE="demo"; shift ;;
  --demo-notify) MODE="demo-notify"; shift ;;
  --demo-outbox) MODE="demo-outbox"; shift ;;
  -h|--help) usage; exit 0 ;;
  "") ;;
  *) echo "unknown arg: $1"; usage; exit 2 ;;
esac

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

metric_get() {
  local addr="$1" selector="$2"
  # Robust Prometheus text format parser (label order independent).
  curl -fsS "$addr" | python3 /dev/fd/3 "$selector" 3<<'PY'
import re,sys

selector = sys.argv[1].strip()

def parse_selector(sel: str):
    m = re.match(r'^([a-zA-Z_:][a-zA-Z0-9_:]*)(\{.*\})?$', sel)
    if not m:
        return sel, {}
    name = m.group(1)
    labels_raw = m.group(2)
    labels = {}
    if labels_raw:
        inner = labels_raw.strip()[1:-1].strip()
        if inner:
            # very small parser: key="value" pairs separated by commas
            parts = [p.strip() for p in inner.split(',') if p.strip()]
            for p in parts:
                k,v = p.split('=',1)
                labels[k.strip()] = v.strip().strip('"')
    return name, labels

name, want = parse_selector(selector)

def parse_labels(lbls: str):
    out = {}
    if not lbls:
        return out
    for p in [x.strip() for x in lbls.split(',') if x.strip()]:
        if '=' not in p:
            continue
        k,v = p.split('=',1)
        out[k.strip()] = v.strip().strip('"')
    return out

val = ''
for line in sys.stdin:
    line = line.strip()
    if not line or line.startswith('#'):
        continue
    if not line.startswith(name):
        continue
    # line: name{...} value OR name value
    m = re.match(r'^([a-zA-Z_:][a-zA-Z0-9_:]*)(\{([^}]*)\})?\s+([-+0-9eE.]+)$', line)
    if not m:
        continue
    lbls = parse_labels(m.group(3) or '')
    ok = True
    for k,v in want.items():
        if lbls.get(k) != v:
            ok = False
            break
    if ok:
        val = m.group(4)

sys.stdout.write(val)
PY
}

metric_wait_gt_zero() {
  local addr="$1" selector="$2" timeout_iters="${3:-40}"
  local v=""
  for i in $(seq 1 "$timeout_iters"); do
    v="$(metric_get "$addr" "$selector" 2>/dev/null || true)"
    if metric_gt_zero "${v:-}"; then
      echo "$v"
      return 0
    fi
    sleep 0.25
  done
  echo "$v"
  return 1
}

metric_gt_zero() {
  local v="$1"
  python3 - <<'PY' "$v"
import sys
try:
    v=float(sys.argv[1])
except Exception:
    sys.exit(1)
sys.exit(0 if v>0 else 2)
PY
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

TMP_ENV_FILE=""
STOPPED_KAFKA=0

PIDFILE_TICKET="/tmp/ticket-service.pid"
PIDFILE_RELAY="/tmp/outbox-relay.pid"
PIDFILE_NOTIFY="/tmp/notification-service.pid"

kill_tree() {
  # Best-effort: stop a process (and its process group if possible).
  local pid="$1"
  [[ -n "$pid" ]] || return 0

  if ! kill -0 "$pid" >/dev/null 2>&1; then
    return 0
  fi

  # Try to signal process group first (works reliably when started with setsid).
  kill -TERM -- "-$pid" >/dev/null 2>&1 || kill -TERM "$pid" >/dev/null 2>&1 || true
  for _ in {1..60}; do
    kill -0 "$pid" >/dev/null 2>&1 || return 0
    sleep 0.1
  done
  kill -KILL -- "-$pid" >/dev/null 2>&1 || kill -KILL "$pid" >/dev/null 2>&1 || true
}

stop_pidfile() {
  local f="$1"
  [[ -f "$f" ]] || return 0
  local pid
  pid="$(cat "$f" 2>/dev/null || true)"
  rm -f "$f" || true
  [[ -n "$pid" ]] || return 0
  kill_tree "$pid"
}

start_service() {
  local name="$1" cmd="$2" logfile="$3" pidfile="$4"
  rm -f "$pidfile" || true
  if command -v setsid >/dev/null 2>&1; then
    # New session => its own process group (PGID=PID), so cleanup can kill the whole group.
    setsid bash -lc "exec $cmd" >"$logfile" 2>&1 &
  else
    bash -lc "exec $cmd" >"$logfile" 2>&1 &
  fi
  local pid=$!
  echo "$pid" >"$pidfile"
  echo "$name pid=$pid"
}

cleanup() {
  echo "cleanup: stopping services..."
  stop_pidfile "$PIDFILE_TICKET"
  stop_pidfile "$PIDFILE_RELAY"
  stop_pidfile "$PIDFILE_NOTIFY"

  if [[ "$STOPPED_KAFKA" -eq 1 ]]; then
    echo "cleanup: starting kafka back"
    docker start "$KAFKA_CONTAINER" >/dev/null 2>&1 || true
  fi

  if [[ -n "$TMP_ENV_FILE" && -f "$TMP_ENV_FILE" ]]; then
    rm -f "$TMP_ENV_FILE" || true
  fi
}
trap cleanup EXIT

if [[ ! -f "$BASE_ENV_FILE" ]]; then
  echo ".env not found -> copying from .env.example"
  cp .env.example "$BASE_ENV_FILE"
fi

ENV_FILE="$BASE_ENV_FILE"

want_notify_demo=0
want_outbox_demo=0
case "$MODE" in
  demo) want_notify_demo=1; want_outbox_demo=1 ;;
  demo-notify) want_notify_demo=1 ;;
  demo-outbox) want_outbox_demo=1 ;;
esac

if [[ "$MODE" != "success" ]]; then
  TMP_ENV_FILE="$(mktemp /tmp/servicedesk-e2e.XXXX.env)"
  cp "$BASE_ENV_FILE" "$TMP_ENV_FILE"

  # Isolate demo runs from any locally running consumers by using a dedicated topic.
  # This prevents another notification-service instance from consuming and marking events as done.
  DEMO_TS="$(date +%s)"
  # Use unique topic per demo run to avoid offset races and old messages.
  DEMO_TOPIC="tickets.events.demo.${DEMO_TS}"
  DEMO_GROUP_ID="notification-service-demo-${DEMO_TS}"
  # New group + unique topic: always start from earliest.
  DEMO_START_OFFSET="first"

  DLQ_TOPIC="${KAFKA_DLQ_TOPIC:-tickets.dlq}"
  OUTBOX_ATTEMPTS="${OUTBOX_RELAY_MAX_ATTEMPTS:-3}"
  NOTIFY_ATTEMPTS="${NOTIFY_MAX_ATTEMPTS:-3}"

  {
    echo ""
    echo "# --- added by scripts/e2e_local.sh ($MODE) ---"
    echo "KAFKA_TOPIC=$DEMO_TOPIC"
    echo "KAFKA_GROUP_ID=$DEMO_GROUP_ID"
    echo "KAFKA_START_OFFSET=$DEMO_START_OFFSET"
    echo "KAFKA_DLQ_TOPIC=$DLQ_TOPIC"
    echo "OUTBOX_RELAY_MAX_ATTEMPTS=$OUTBOX_ATTEMPTS"
    echo "NOTIFY_MAX_ATTEMPTS=$NOTIFY_ATTEMPTS"
    if [[ "$want_notify_demo" -eq 1 ]]; then
      echo "NOTIFY_FORCE_FAIL=1"
      # Make demo deterministic: fail ANY consumed event (event_type filter is optional and can be flaky if env contains hidden whitespace).
      echo "NOTIFY_FORCE_FAIL_EVENT_TYPE="
    fi
  } >> "$TMP_ENV_FILE"

  ENV_FILE="$TMP_ENV_FILE"
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

# In success mode, don't inherit demo flags from the parent shell.
# (Very common cause of "processed_events not done" is leftover NOTIFY_FORCE_FAIL=1 or demo topic/group in env.)
if [[ "$MODE" == "success" && "${E2E_NO_SANITIZE:-0}" != "1" ]]; then
  export NOTIFY_FORCE_FAIL=0
  unset NOTIFY_FORCE_FAIL_EVENT_TYPE
  export KAFKA_START_OFFSET="first"
  export KAFKA_GROUP_ID="notification-service"
  export KAFKA_TOPIC="tickets.events"
fi

validate_env

echo "=== preflight: stop any locally running go-run services (to avoid ghost consumers) ==="
pkill -f "go run ./cmd/ticket-service" >/dev/null 2>&1 || true
pkill -f "go run ./cmd/outbox-relay" >/dev/null 2>&1 || true
pkill -f "go run ./cmd/notification-service" >/dev/null 2>&1 || true

echo "=== preflight: check ports are free (8080/9090/9091) ==="
if command -v ss >/dev/null 2>&1; then
  if ss -lntp 2>/dev/null | grep -E ':(8080|9090|9091)\b' >/dev/null 2>&1; then
    echo "ERROR: one of ports 8080/9090/9091 is already in use."
    ss -lntp 2>/dev/null | grep -E ':(8080|9090|9091)\b' || true
    echo "Stop the processes above and re-run."
    exit 2
  fi
fi

echo "=== compose up ==="
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d

wait_port "127.0.0.1" "5432" "postgres"
wait_postgres_ready

if docker ps --format '{{.Names}}' | grep -q "$KAFKA_CONTAINER"; then
  wait_port "127.0.0.1" "29092" "kafka(host listener)"
fi

TOPIC="${KAFKA_TOPIC:-tickets.events}"
DLQ_TOPIC="${KAFKA_DLQ_TOPIC:-}"

echo "=== ensure topic exists ==="
docker exec -i "$KAFKA_CONTAINER" bash -lc \
  "/opt/kafka/bin/kafka-topics.sh --bootstrap-server $KAFKA_BOOTSTRAP_IN_CONTAINER \
   --create --if-not-exists --topic $TOPIC --partitions 1 --replication-factor 1" >/dev/null 2>&1 || true

if [[ -n "$DLQ_TOPIC" ]]; then
  echo "=== ensure DLQ topic exists ==="
  docker exec -i "$KAFKA_CONTAINER" bash -lc \
    "/opt/kafka/bin/kafka-topics.sh --bootstrap-server $KAFKA_BOOTSTRAP_IN_CONTAINER \
     --create --if-not-exists --topic $DLQ_TOPIC --partitions 1 --replication-factor 1" >/dev/null 2>&1 || true
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
start_service "ticket-service" "go run ./cmd/ticket-service" "/tmp/ticket-service.log" "$PIDFILE_TICKET"

echo "=== start outbox-relay ==="
start_service "outbox-relay" "go run ./cmd/outbox-relay" "/tmp/outbox-relay.log" "$PIDFILE_RELAY"

echo "=== start notification-service ==="
start_service "notification-service" "go run ./cmd/notification-service" "/tmp/notification-service.log" "$PIDFILE_NOTIFY"

echo "wait: ticket-service healthz"
for i in {1..60}; do
  if curl -fsS "$TICKET_ADDR/healthz" >/dev/null 2>&1; then break; fi
  sleep 0.5
done


echo "wait: outbox-relay metrics"
for i in {1..60}; do
  if curl -fsS "$METRICS_ADDR_RELAY" >/dev/null 2>&1; then break; fi
  sleep 0.5
done
curl -fsS "$METRICS_ADDR_RELAY" >/dev/null 2>&1 || {
  echo "FAIL: outbox-relay didn't start (check /tmp/outbox-relay.log)"
  tail -n 200 /tmp/outbox-relay.log || true
  exit 1
}

echo "wait: notification-service metrics"
for i in {1..60}; do
  if curl -fsS "$METRICS_ADDR_NOTIFY" >/dev/null 2>&1; then break; fi
  sleep 0.5
done
curl -fsS "$METRICS_ADDR_NOTIFY" >/dev/null 2>&1 || {
  echo "FAIL: notification-service didn't start (check /tmp/notification-service.log)"
  tail -n 200 /tmp/notification-service.log || true
  exit 1
}

create_ticket() {
  local title="$1" desc="$2" req_id="$3"
  curl -fsS -X POST "$TICKET_ADDR/tickets" \
    -H 'Content-Type: application/json' \
    -H "X-Request-Id: $req_id" \
    -d "{\"title\":\"$title\",\"description\":\"$desc\"}"
}

wait_outbox_status() {
  local ticket_id="$1" want="$2" timeout_iters="${3:-80}"
  local row event_id status
  event_id=""
  for i in $(seq 1 "$timeout_iters"); do
    row="$(docker exec -i "$PG_CONTAINER" psql -U "${POSTGRES_USER:-servicedesk}" -d "${POSTGRES_DB:-servicedesk}" -Atc \
      "select event_id||'|'||status from outbox where aggregate_id='$ticket_id' order by created_at desc limit 1;" 2>/dev/null || true)"

    if [[ -n "$row" ]]; then
      event_id="${row%%|*}"
      status="${row##*|}"
      if [[ "$status" == "$want" ]]; then
        echo "$event_id"
        return 0
      fi
      echo "outbox: event_id=$event_id status=$status (waiting for $want)" >&2
    else
      echo "outbox: no row yet (waiting)" >&2
    fi
    sleep 0.5
  done
  echo ""
  return 1
}

wait_processed_status() {
  local event_id="$1" want="$2" timeout_iters="${3:-80}"
  local status
  for i in $(seq 1 "$timeout_iters"); do
    status="$(docker exec -i "$PG_CONTAINER" psql -U "${POSTGRES_USER:-servicedesk}" -d "${POSTGRES_DB:-servicedesk}" -Atc \
      "select status from processed_events where event_id='$event_id' limit 1;" 2>/dev/null || true)"
    if [[ "$status" == "$want" ]]; then
      return 0
    fi
    sleep 0.5
  done
  return 1
}

run_happy_path() {
  echo "=== create ticket ==="
  REQ_ID="rid-e2e-$(date +%s)"
  RESP="$(create_ticket "E2E ticket" "created by scripts/e2e_local.sh" "$REQ_ID")"

  TICKET_ID="$(json_get "$RESP" "id")"
  if [[ -z "$TICKET_ID" ]]; then
    echo "failed to parse ticket id from response: $RESP"
    echo "ticket-service log: /tmp/ticket-service.log"
    exit 1
  fi
  echo "ticket_id=$TICKET_ID"

  echo "=== wait outbox sent ==="
  EVENT_ID="$(wait_outbox_status "$TICKET_ID" "sent" 80)" || {
    echo "FAIL: outbox event not sent for ticket_id=$TICKET_ID"
    echo "relay log: /tmp/outbox-relay.log"
    exit 1
  }
  echo "outbox: event_id=$EVENT_ID status=sent"

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
  if ! wait_processed_status "$EVENT_ID" "done" 80; then
    echo "FAIL: processed_events not done for event_id=$EVENT_ID"
    echo "--- diag(processed_events) ---"
    ./scripts/diag.sh processed || true
    echo "--- tail notification-service ---"
    tail -n 200 /tmp/notification-service.log || true
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
}

run_demo_notify() {
  if [[ -z "$DLQ_TOPIC" ]]; then
    echo "FAIL: demo-notify requires KAFKA_DLQ_TOPIC (it is auto-set when running with --demo/--demo-notify)"
    exit 2
  fi

  echo "=== DEMO: notify -> retries -> processed_events=failed + DLQ ==="
  REQ_ID="rid-demo-notify-$(date +%s)"
  RESP="$(create_ticket "DEMO notify fail" "should go to DLQ" "$REQ_ID")"
  TICKET_ID="$(json_get "$RESP" "id")"
  [[ -n "$TICKET_ID" ]] || { echo "FAIL: can't parse ticket id: $RESP"; exit 1; }

  echo "ticket_id=$TICKET_ID"

  echo "=== wait outbox sent ==="
  EVENT_ID="$(wait_outbox_status "$TICKET_ID" "sent" 80)" || {
    echo "FAIL: outbox event not sent for ticket_id=$TICKET_ID"
    exit 1
  }
  echo "event_id=$EVENT_ID"

  echo "=== wait processed_events failed ==="
  if ! wait_processed_status "$EVENT_ID" "failed" 120; then
    echo "FAIL: processed_events not failed for event_id=$EVENT_ID"
    echo "--- diag(processed_events) ---"
    ./scripts/diag.sh processed || true
    echo "--- tail notification-service ---"
    tail -n 200 /tmp/notification-service.log || true
    echo "logs:"
    echo "  /tmp/ticket-service.log"
    echo "  /tmp/outbox-relay.log"
    echo "  /tmp/notification-service.log"
    exit 1
  fi

  echo "=== peek DLQ for event_id (best effort) ==="
  dlq_out="$(docker exec -i "$KAFKA_CONTAINER" bash -lc \
    "/opt/kafka/bin/kafka-console-consumer.sh \
     --bootstrap-server $KAFKA_BOOTSTRAP_IN_CONTAINER \
     --topic $DLQ_TOPIC --from-beginning --max-messages 50 --timeout-ms $KAFKA_PEEK_TIMEOUT_MS" 2>/dev/null || true)"

  if echo "$dlq_out" | grep -q "$EVENT_ID"; then
    echo "DLQ: found event_id=$EVENT_ID in $DLQ_TOPIC"
  else
    echo "DLQ: could not confirm event_id in peek (ok if timeout)"
  fi

  echo "=== check notify metrics increased ==="
  v1="$(metric_wait_gt_zero "$METRICS_ADDR_NOTIFY" 'notify_errors_total{event_type="ticket.created",reason="forced"}' 60)" || {
    echo "FAIL: expected notify_errors_total{event_type=\"ticket.created\",reason=\"forced\"} > 0 (got '$v1')"
    echo "--- notify metrics snippet ---"
    curl -fsS "$METRICS_ADDR_NOTIFY" | grep -E '^notify_(errors_total|processed_total)' | tail -n 50 || true
    exit 1
  }
  v2="$(metric_wait_gt_zero "$METRICS_ADDR_NOTIFY" 'notify_processed_total{event_type="ticket.created",status="dead"}' 60)" || {
    echo "FAIL: expected notify_processed_total{event_type=\"ticket.created\",status=\"dead\"} > 0 (got '$v2')"
    echo "--- notify metrics snippet ---"
    curl -fsS "$METRICS_ADDR_NOTIFY" | grep -E '^notify_(errors_total|processed_total)' | tail -n 50 || true
    exit 1
  }

  echo "✅ DEMO notify OK: event_id=$EVENT_ID (processed_events=failed, DLQ best-effort)"
}

run_demo_outbox() {
  echo "=== DEMO: outbox-relay -> retries -> outbox=failed (Kafka down) ==="

  echo "stop kafka container: $KAFKA_CONTAINER"
  docker stop "$KAFKA_CONTAINER" >/dev/null
  STOPPED_KAFKA=1

  REQ_ID="rid-demo-outbox-$(date +%s)"
  RESP="$(create_ticket "DEMO outbox fail" "kafka is down -> outbox should become failed" "$REQ_ID")"
  TICKET_ID="$(json_get "$RESP" "id")"
  [[ -n "$TICKET_ID" ]] || { echo "FAIL: can't parse ticket id: $RESP"; exit 1; }
  echo "ticket_id=$TICKET_ID"

  echo "=== wait outbox failed ==="
  EVENT_ID="$(wait_outbox_status "$TICKET_ID" "failed" 160)" || {
    echo "FAIL: outbox not failed for ticket_id=$TICKET_ID (check relay log /tmp/outbox-relay.log)"
    exit 1
  }
  echo "outbox: event_id=$EVENT_ID status=failed"

  echo "=== check relay metrics increased ==="
  v="$(metric_wait_gt_zero "$METRICS_ADDR_RELAY" 'outbox_dead_total{event_type="ticket.created"}' 60)" || {
    echo "FAIL: expected outbox_dead_total{event_type=\"ticket.created\"} > 0 (got '$v')"
    echo "--- relay metrics snippet ---"
    curl -fsS "$METRICS_ADDR_RELAY" | grep -E '^outbox_(dead_total|failed_total|published_total)' | tail -n 50 || true
    exit 1
  }

  echo "start kafka back"
  docker start "$KAFKA_CONTAINER" >/dev/null
  STOPPED_KAFKA=0

  echo "✅ DEMO outbox OK: event_id=$EVENT_ID (outbox=failed, dead metric)"
}

case "$MODE" in
  success)
    run_happy_path
    ;;
  demo-notify)
    run_demo_notify
    ;;
  demo-outbox)
    run_demo_outbox
    ;;
  demo)
    run_demo_notify
    run_demo_outbox
    echo "✅ DEMO complete"
    ;;
esac
