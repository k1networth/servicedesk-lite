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

	reg := prometheus.NewRegistry()
	met := outbox.NewMetrics(reg)

	// metrics server
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		log.Info("metrics_listen", slog.String("addr", metricsAddr))
		_ = http.ListenAndServe(metricsAddr, mux)
	}()

	log.Info("relay_start",
		slog.Int("batch_size", batchSize),
		slog.String("poll_interval", pollInterval.String()),
		slog.String("processing_timeout", processingTimeout.String()),
		slog.String("topic", topic),
	)

	t := time.NewTicker(pollInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("relay_shutdown")
			return
		case <-t.C:
			if err := tick(ctx, log, store, producer, met, batchSize, processingTimeout); err != nil {
				log.Error("relay_tick_failed", slog.String("err", err.Error()))
			}
		}
	}
}

func tick(ctx context.Context, log *slog.Logger, store *outbox.Store, producer *kafkax.Producer, met *outbox.Metrics, batchSize int, processingTimeout time.Duration) error {
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
			_ = store.MarkFailed(ctx, e.ID, time.Now().Add(backoff(e.Attempts)), "marshal: "+err.Error())
			met.FailedTotal.WithLabelValues(e.EventType).Inc()
			continue
		}

		if err := producer.Produce(ctx, []byte(e.AggregateID), b, 5*time.Second); err != nil {
			next := time.Now().Add(backoff(e.Attempts))
			_ = store.MarkFailed(ctx, e.ID, next, "kafka: "+err.Error())
			met.FailedTotal.WithLabelValues(e.EventType).Inc()
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
