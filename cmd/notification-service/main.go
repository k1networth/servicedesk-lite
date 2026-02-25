package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/k1networth/servicedesk-lite/internal/notify"
	"github.com/k1networth/servicedesk-lite/internal/shared/config"
	"github.com/k1networth/servicedesk-lite/internal/shared/db"
	"github.com/k1networth/servicedesk-lite/internal/shared/env"
	"github.com/k1networth/servicedesk-lite/internal/shared/events"
	"github.com/k1networth/servicedesk-lite/internal/shared/kafkax"
	"github.com/k1networth/servicedesk-lite/internal/shared/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const appName = "notification-service"

func main() {
	cfg := config.Load()
	log := logger.New(appName, cfg.AppEnv)

	dbURL := env.String("DATABASE_URL", cfg.DatabaseURL)
	if dbURL == "" {
		log.Error("config_error", slog.String("err", "DATABASE_URL is empty"))
		os.Exit(2)
	}

	brokers := env.StringsCSV("KAFKA_BROKERS", []string{"localhost:9092"})
	topic := env.String("KAFKA_TOPIC", "tickets.events")
	groupID := env.String("KAFKA_GROUP_ID", appName)
	startOffset := env.String("KAFKA_START_OFFSET", "last")
	dlqTopic := env.String("KAFKA_DLQ_TOPIC", "")
	maxAttempts := env.Int("NOTIFY_MAX_ATTEMPTS", 10)
	forceFail := env.Bool("NOTIFY_FORCE_FAIL", false)
	forceFailEventType := env.String("NOTIFY_FORCE_FAIL_EVENT_TYPE", "")
	metricsAddr := env.String("METRICS_ADDR", ":9091")

	// Debug-only: log raw env values to avoid "0 treated as true" style bugs.
	// IMPORTANT: logic must use parsed values above.
	forceFailRaw := os.Getenv("NOTIFY_FORCE_FAIL")
	maxAttemptsRaw := os.Getenv("NOTIFY_MAX_ATTEMPTS")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pg, err := db.OpenPostgres(ctx, db.PostgresConfig{DatabaseURL: dbURL})
	if err != nil {
		log.Error("db_open_failed", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer func() { _ = pg.Close() }()

	store := notify.NewStore(pg)
	consumer := kafkax.NewConsumer(kafkax.ConsumerConfig{Brokers: brokers, Topic: topic, GroupID: groupID, StartOffset: startOffset})
	defer func() { _ = consumer.Close() }()

	var dlqProducer *kafkax.Producer
	if dlqTopic != "" {
		dlqProducer = kafkax.NewProducer(kafkax.ProducerConfig{
			Brokers:      brokers,
			Topic:        dlqTopic,
			ClientID:     appName + "-dlq",
			WriteTimeout: 5 * time.Second,
		})
		defer func() { _ = dlqProducer.Close() }()
	}

	reg := prometheus.NewRegistry()
	// Standard collectors so /metrics is never empty (useful for demos & Grafana).
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(collectors.NewBuildInfoCollector())
	processed := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "notify_processed_total", Help: "Processed events."}, []string{"event_type", "status"})
	errors := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "notify_errors_total", Help: "Notification service errors."}, []string{"event_type", "reason"})
	reg.MustRegister(processed)
	reg.MustRegister(errors)
	// Kafka loop observability.
	fetched := prometheus.NewCounter(prometheus.CounterOpts{Name: "notify_fetched_total", Help: "Fetched Kafka messages."})
	committed := prometheus.NewCounter(prometheus.CounterOpts{Name: "notify_committed_total", Help: "Committed Kafka messages."})
	lastProcessedUnix := prometheus.NewGauge(prometheus.GaugeOpts{Name: "notify_last_processed_unix", Help: "Unix timestamp of last processed (committed) message."})
	reg.MustRegister(fetched)
	reg.MustRegister(committed)
	reg.MustRegister(lastProcessedUnix)

	// Pre-create common labelsets so they show up even before first message.
	processed.WithLabelValues("ticket.created", "ok").Add(0)
	processed.WithLabelValues("ticket.created", "dead").Add(0)
	errors.WithLabelValues("ticket.created", "db_start").Add(0)
	errors.WithLabelValues("ticket.created", "unmarshal").Add(0)

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})
		mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ready"))
		})
		mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		log.Info("metrics_listen", slog.String("addr", metricsAddr))
		_ = http.ListenAndServe(metricsAddr, mux)
	}()

	log.Info(
		"consumer_start",
		slog.String("topic", topic),
		slog.String("group_id", groupID),
		slog.String("start_offset", startOffset),
		slog.String("dlq_topic", dlqTopic),
		slog.Int("max_attempts", maxAttempts),
		slog.String("max_attempts_raw", maxAttemptsRaw),
		slog.Bool("force_fail", forceFail),
		slog.String("force_fail_raw", forceFailRaw),
		slog.String("force_fail_event_type", forceFailEventType),
	)

	for {
		select {
		case <-ctx.Done():
			log.Info("consumer_shutdown")
			return
		default:
			msg, err := consumer.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					continue
				}
				log.Error("kafka_fetch_failed", slog.String("err", err.Error()))
				time.Sleep(300 * time.Millisecond)
				continue
			}

			fetched.Inc()
			// Log fetch with best-effort envelope info (helps prove Kafka -> consumer in demos).
			{
				var env events.Envelope
				if err := json.Unmarshal(msg.Value, &env); err == nil {
					log.Info("kafka_message_fetched",
						slog.Int("partition", msg.Partition),
						slog.Int64("offset", msg.Offset),
						slog.String("event_id", env.EventID),
						slog.String("event_type", env.EventType),
						slog.String("aggregate_id", env.AggregateID),
					)
				} else {
					log.Warn("kafka_message_fetched_unmarshal_failed",
						slog.Int("partition", msg.Partition),
						slog.Int64("offset", msg.Offset),
						slog.String("err", err.Error()),
					)
				}
			}

			evType := "unknown"
			attempt := 0

			// Important: kafka-go Reader (with GroupID) will continue fetching newer messages even if
			// we don't commit. So "retry by not committing" does NOT re-deliver the same message.
			// To implement retries deterministically, we retry handling the SAME message in-process
			// and commit only after success or after moving it to failed/DLQ.
			for {
				err = handleMessage(ctx, log, store, msg.Value, forceFail, forceFailEventType, &evType, &attempt)
				if err == nil {
					processed.WithLabelValues(evType, "ok").Inc()
					if err := consumer.CommitMessages(ctx, msg); err != nil {
						log.Error("kafka_commit_failed", slog.String("err", err.Error()))
					} else {
						committed.Inc()
						lastProcessedUnix.SetToCurrentTime()
						log.Info("kafka_commit_ok", slog.Int("partition", msg.Partition), slog.Int64("offset", msg.Offset))
					}
					break
				}

				reason := classify(err)
				errors.WithLabelValues(evType, reason).Inc()
				log.Error("message_handle_failed", slog.String("reason", reason), slog.Int("attempt", attempt), slog.String("err", err.Error()))

				// Non-retryable: can't decode the message.
				if reason == "unmarshal" {
					if dlqProducer != nil {
						_ = dlqProducer.Produce(ctx, msg.Key, wrapDLQ(msg.Value, err), 5*time.Second)
					}
					if err := consumer.CommitMessages(ctx, msg); err != nil {
						log.Error("kafka_commit_failed", slog.String("err", err.Error()))
					} else {
						committed.Inc()
						lastProcessedUnix.SetToCurrentTime()
						log.Info("kafka_commit_ok", slog.Int("partition", msg.Partition), slog.Int64("offset", msg.Offset))
					}
					processed.WithLabelValues(evType, "dead").Inc()
					break
				}

				// No infinite loops: after max attempts, send to DLQ (best-effort), mark failed and commit.
				if maxAttempts > 0 && attempt >= maxAttempts {
					if dlqProducer != nil {
						_ = dlqProducer.Produce(ctx, msg.Key, wrapDLQ(msg.Value, err), 5*time.Second)
					}
					_ = store.MarkDead(ctx, extractEventID(msg.Value), err.Error())
					if err := consumer.CommitMessages(ctx, msg); err != nil {
						log.Error("kafka_commit_failed", slog.String("err", err.Error()))
					} else {
						committed.Inc()
						lastProcessedUnix.SetToCurrentTime()
						log.Info("kafka_commit_ok", slog.Int("partition", msg.Partition), slog.Int64("offset", msg.Offset))
					}
					processed.WithLabelValues(evType, "dead").Inc()
					break
				}

				// Backoff to avoid busy-loop while retrying same message.
				time.Sleep(backoff(attempt))
			}
		}
	}
}

func handleMessage(ctx context.Context, log *slog.Logger, store *notify.Store, value []byte, forceFail bool, forceFailEventType string, eventTypeOut *string, attemptOut *int) error {
	var env events.Envelope
	if err := json.Unmarshal(value, &env); err != nil {
		return wrap("unmarshal", err)
	}
	*eventTypeOut = env.EventType

	// Robust match (avoid invisible whitespace / case issues in demo env vars).
	matchForceFail := false
	if forceFail {
		want := strings.TrimSpace(forceFailEventType)
		got := strings.TrimSpace(env.EventType)
		matchForceFail = (want == "" || strings.EqualFold(want, got))
	}

	shouldProcess, attempts, _, err := store.StartProcessing(ctx, notify.ProcessedEvent{
		EventID:     env.EventID,
		EventType:   env.EventType,
		Aggregate:   env.Aggregate,
		AggregateID: env.AggregateID,
		Payload:     env.Payload,
	})
	if err != nil {
		return wrap("db_start", err)
	}
	*attemptOut = attempts
	if !shouldProcess {
		log.Info("event_skip_done", slog.String("event_id", env.EventID), slog.String("event_type", env.EventType))
		return nil
	}

	log.Info("notify_event", slog.String("event_id", env.EventID), slog.String("event_type", env.EventType), slog.String("aggregate_id", env.AggregateID))

	if matchForceFail {
		log.Warn(
			"force_fail_hit",
			slog.String("event_id", env.EventID),
			slog.String("event_type", env.EventType),
			slog.String("force_fail_event_type", strings.TrimSpace(forceFailEventType)),
		)
		_ = store.MarkFailed(ctx, env.EventID, "forced failure")
		return wrap("forced", errors.New("forced failure"))
	}

	if err := store.MarkDone(ctx, env.EventID); err != nil {
		_ = store.MarkFailed(ctx, env.EventID, err.Error())
		return wrap("db_done", err)
	}
	return nil
}

type taggedErr struct {
	tag string
	err error
}

func (e taggedErr) Error() string { return e.tag + ": " + e.err.Error() }
func (e taggedErr) Unwrap() error { return e.err }

func wrap(tag string, err error) error { return taggedErr{tag: tag, err: err} }

func classify(err error) string {
	var te taggedErr
	if ok := errors.As(err, &te); ok {
		return te.tag
	}
	return "unknown"
}

func extractEventID(value []byte) string {
	var env events.Envelope
	_ = json.Unmarshal(value, &env)
	return env.EventID
}

func wrapDLQ(value []byte, err error) []byte {
	dlq := struct {
		Error string          `json:"error"`
		Value json.RawMessage `json:"value"`
	}{
		Error: err.Error(),
		Value: value,
	}
	b, _ := json.Marshal(dlq)
	return b
}

func backoff(attempt int) time.Duration {
	if attempt <= 1 {
		return 200 * time.Millisecond
	}
	d := time.Duration(1<<uint(min(attempt-1, 6))) * time.Second
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
