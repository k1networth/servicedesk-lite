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

	"github.com/k1networth/servicedesk-lite/internal/notify"
	"github.com/k1networth/servicedesk-lite/internal/shared/config"
	"github.com/k1networth/servicedesk-lite/internal/shared/db"
	"github.com/k1networth/servicedesk-lite/internal/shared/env"
	"github.com/k1networth/servicedesk-lite/internal/shared/events"
	"github.com/k1networth/servicedesk-lite/internal/shared/kafkax"
	"github.com/k1networth/servicedesk-lite/internal/shared/logger"
	"github.com/prometheus/client_golang/prometheus"
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
	metricsAddr := env.String("METRICS_ADDR", ":9091")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pg, err := db.OpenPostgres(ctx, db.PostgresConfig{DatabaseURL: dbURL})
	if err != nil {
		log.Error("db_open_failed", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer func() { _ = pg.Close() }()

	store := notify.NewStore(pg)
	consumer := kafkax.NewConsumer(kafkax.ConsumerConfig{Brokers: brokers, Topic: topic, GroupID: groupID})
	defer func() { _ = consumer.Close() }()

	reg := prometheus.NewRegistry()
	processed := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "notify_processed_total", Help: "Processed events."}, []string{"event_type", "status"})
	reg.MustRegister(processed)

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		log.Info("metrics_listen", slog.String("addr", metricsAddr))
		_ = http.ListenAndServe(metricsAddr, mux)
	}()

	log.Info("consumer_start", slog.String("topic", topic), slog.String("group_id", groupID))

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

			statusLabel := "ok"
			evType := "unknown"

			err = handleMessage(ctx, log, store, msg.Value, &evType)
			if err != nil {
				statusLabel = "error"
				log.Error("message_handle_failed", slog.String("err", err.Error()))
			}

			processed.WithLabelValues(evType, statusLabel).Inc()

			if err != nil {
				continue
			}
			if err := consumer.CommitMessages(ctx, msg); err != nil {
				log.Error("kafka_commit_failed", slog.String("err", err.Error()))
				continue
			}
		}
	}
}

func handleMessage(ctx context.Context, log *slog.Logger, store *notify.Store, value []byte, eventTypeOut *string) error {
	var env events.Envelope
	if err := json.Unmarshal(value, &env); err != nil {
		return err
	}
	*eventTypeOut = env.EventType

	shouldProcess, err := store.StartProcessing(ctx, notify.ProcessedEvent{
		EventID:     env.EventID,
		EventType:   env.EventType,
		Aggregate:   env.Aggregate,
		AggregateID: env.AggregateID,
		Payload:     env.Payload,
	})
	if err != nil {
		return err
	}
	if !shouldProcess {
		log.Info("event_skip_done", slog.String("event_id", env.EventID), slog.String("event_type", env.EventType))
		return nil
	}

	log.Info("notify_event", slog.String("event_id", env.EventID), slog.String("event_type", env.EventType), slog.String("aggregate_id", env.AggregateID))

	if err := store.MarkDone(ctx, env.EventID); err != nil {
		_ = store.MarkFailed(ctx, env.EventID, err.Error())
		return err
	}
	return nil
}
