package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/k1networth/servicedesk-lite/internal/outbox"
	"github.com/k1networth/servicedesk-lite/internal/shared/config"
	"github.com/k1networth/servicedesk-lite/internal/shared/db"
	"github.com/k1networth/servicedesk-lite/internal/shared/env"
	"github.com/k1networth/servicedesk-lite/internal/shared/events"
	"github.com/k1networth/servicedesk-lite/internal/shared/kafkax"
	"github.com/k1networth/servicedesk-lite/internal/shared/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const appName = "outbox-relay"

func main() {
	cfg := config.Load()
	log := logger.New(appName, cfg.AppEnv)

	// Required
	dbURL := env.String("DATABASE_URL", cfg.DatabaseURL)
	if dbURL == "" {
		log.Error("config_error", slog.String("err", "DATABASE_URL is empty"))
		os.Exit(2)
	}

	brokers := env.StringsCSV("KAFKA_BROKERS", []string{"localhost:9092"})
	topic := env.String("KAFKA_TOPIC", "tickets.events")
	clientID := env.String("KAFKA_CLIENT_ID", appName)

	batchSize := env.Int("OUTBOX_RELAY_BATCH_SIZE", 50)
	pollInterval := env.Duration("OUTBOX_RELAY_POLL_INTERVAL", 500*time.Millisecond)
	processingTimeout := env.Duration("OUTBOX_RELAY_PROCESSING_TIMEOUT", 30*time.Second)
	maxAttempts := env.Int("OUTBOX_RELAY_MAX_ATTEMPTS", 10)
	dlqTopic := env.String("KAFKA_DLQ_TOPIC", "")
	metricsAddr := env.String("METRICS_ADDR", ":9090")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pg, err := db.OpenPostgres(ctx, db.PostgresConfig{DatabaseURL: dbURL})
	if err != nil {
		log.Error("db_open_failed", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer func() { _ = pg.Close() }()

	store := outbox.NewStore(pg)
	producer := kafkax.NewProducer(kafkax.ProducerConfig{
		Brokers:      brokers,
		Topic:        topic,
		ClientID:     clientID,
		WriteTimeout: 5 * time.Second,
	})
	defer func() { _ = producer.Close() }()

	var dlqProducer *kafkax.Producer
	if dlqTopic != "" {
		dlqProducer = kafkax.NewProducer(kafkax.ProducerConfig{
			Brokers:      brokers,
			Topic:        dlqTopic,
			ClientID:     clientID + "-dlq",
			WriteTimeout: 5 * time.Second,
		})
		defer func() { _ = dlqProducer.Close() }()
	}

	reg := prometheus.NewRegistry()
	met := outbox.NewMetrics(reg)

	// metrics server
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

	log.Info("relay_start",
		slog.Int("batch_size", batchSize),
		slog.String("poll_interval", pollInterval.String()),
		slog.String("processing_timeout", processingTimeout.String()),
		slog.Int("max_attempts", maxAttempts),
		slog.String("topic", topic),
		slog.String("dlq_topic", dlqTopic),
	)

	t := time.NewTicker(pollInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("relay_shutdown")
			return
		case <-t.C:
			if err := tick(ctx, log, store, producer, dlqProducer, met, batchSize, processingTimeout, maxAttempts); err != nil {
				log.Error("relay_tick_failed", slog.String("err", err.Error()))
			}
		}
	}
}

func tick(ctx context.Context, log *slog.Logger, store *outbox.Store, producer *kafkax.Producer, dlqProducer *kafkax.Producer, met *outbox.Metrics, batchSize int, processingTimeout time.Duration, maxAttempts int) error {
	if _, err := store.ResetStuck(ctx, processingTimeout); err != nil {
		return err
	}

	if lag, err := store.LagSeconds(ctx); err == nil {
		met.LagSeconds.Set(lag)
	}

	rows, err := store.ClaimPending(ctx, batchSize)
	if err != nil {
		return err
	}

	for _, e := range rows {
		var ridHolder struct {
			RequestID string `json:"request_id"`
		}
		_ = json.Unmarshal(e.Payload, &ridHolder)

		env := events.Envelope{
			EventID:     e.EventID,
			EventType:   e.EventType,
			OccurredAt:  e.CreatedAt,
			Aggregate:   e.Aggregate,
			AggregateID: e.AggregateID,
			RequestID:   ridHolder.RequestID,
			Payload:     e.Payload,
		}
		b, err := json.Marshal(env)
		if err != nil {
			dead := maybeDead(ctx, log, store, dlqProducer, met, e, env, maxAttempts, "marshal: "+err.Error())
			if !dead {
				_ = store.MarkFailed(ctx, e.ID, time.Now().Add(backoff(e.Attempts)), "marshal: "+err.Error())
				met.FailedTotal.WithLabelValues(e.EventType).Inc()
			}
			continue
		}

		if err := producer.Produce(ctx, []byte(e.AggregateID), b, 5*time.Second); err != nil {
			log.Warn("publish_failed",
				slog.Int64("outbox_id", e.ID),
				slog.String("event_type", e.EventType),
				slog.Int("attempt", e.Attempts),
				slog.String("err", err.Error()),
			)
			dead := maybeDead(ctx, log, store, dlqProducer, met, e, env, maxAttempts, "kafka: "+err.Error())
			if !dead {
				next := time.Now().Add(backoff(e.Attempts))
				_ = store.MarkFailed(ctx, e.ID, next, "kafka: "+err.Error())
				met.FailedTotal.WithLabelValues(e.EventType).Inc()
			}
			continue
		}

		if err := store.MarkSent(ctx, e.ID); err != nil {
			log.Error("mark_sent_failed", slog.Int64("outbox_id", e.ID), slog.String("err", err.Error()))
		}
		met.PublishedTotal.WithLabelValues(e.EventType).Inc()
	}

	return nil
}

func backoff(attempt int) time.Duration {
	if attempt <= 1 {
		return 500 * time.Millisecond
	}
	d := time.Duration(1<<uint(min(attempt-1, 6))) * time.Second
	if d > 60*time.Second {
		d = 60 * time.Second
	}
	return d
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maybeDead(ctx context.Context, log *slog.Logger, store *outbox.Store, dlqProducer *kafkax.Producer, met *outbox.Metrics, e outbox.Event, env events.Envelope, maxAttempts int, errMsg string) bool {
	if maxAttempts > 0 && e.Attempts >= maxAttempts {
		// Best-effort DLQ publish; even if it fails we still mark DB row as failed to avoid infinite retries.
		if dlqProducer != nil {
			dlq := struct {
				Error    string          `json:"error"`
				Envelope events.Envelope `json:"envelope"`
			}{
				Error:    errMsg,
				Envelope: env,
			}
			if b, err := json.Marshal(dlq); err == nil {
				_ = dlqProducer.Produce(ctx, []byte(e.AggregateID), b, 5*time.Second)
			}
		}

		if err := store.MarkDead(ctx, e.ID, errMsg); err != nil {
			log.Error("mark_dead_failed", slog.Int64("outbox_id", e.ID), slog.String("err", err.Error()))
		}
		met.DeadTotal.WithLabelValues(e.EventType).Inc()
		return true
	}
	return false
}
