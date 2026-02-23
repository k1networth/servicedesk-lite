package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/k1networth/servicedesk-lite/internal/outbox"
	"github.com/k1networth/servicedesk-lite/internal/shared/config"
	"github.com/k1networth/servicedesk-lite/internal/shared/db"
	"github.com/k1networth/servicedesk-lite/internal/shared/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const appName = "outbox-relay"

func main() {
	cfg := config.Load()
	log := logger.New(appName, cfg.AppEnv)

	if cfg.DatabaseURL == "" {
		log.Error("DATABASE_URL is empty")
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pg, err := db.OpenPostgres(ctx, db.PostgresConfig{DatabaseURL: cfg.DatabaseURL})
	if err != nil {
		log.Error("db_open_failed", slog.String("err", err.Error()))
		return
	}
	defer func() {
		if err := pg.Close(); err != nil {
			log.Error("db_close_failed", slog.String("err", err.Error()))
		}
	}()

	repo := outbox.NewPostgresRepo(pg)

	// --- Metrics ---
	reg := prometheus.NewRegistry()
	m := newRelayMetrics(reg)

	metricsSrv := &http.Server{
		Addr:              cfg.MetricsAddr,
		Handler:           promhttp.HandlerFor(reg, promhttp.HandlerOpts{}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("metrics_listen", slog.String("addr", metricsSrv.Addr))
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("metrics_server_error", slog.String("err", err.Error()))
		}
	}()

	log.Info("relay_start",
		slog.Int("batch_size", cfg.OutboxBatchSize),
		slog.String("poll_interval", cfg.OutboxPollInterval.String()),
		slog.String("processing_timeout", cfg.OutboxProcessingTimeout.String()),
	)

	ticker := time.NewTicker(cfg.OutboxPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = metricsSrv.Shutdown(shutdownCtx)
			log.Info("relay_shutdown")
			return
		case <-ticker.C:
			m.pollsTotal.Inc()

			if n, err := repo.RequeueStuck(ctx, cfg.OutboxProcessingTimeout); err != nil {
				m.requeueErrorsTotal.Inc()
				log.Error("outbox_requeue_failed", slog.String("err", err.Error()))
			} else if n > 0 {
				m.requeuedTotal.Add(float64(n))
				log.Warn("outbox_requeued_stuck", slog.Int64("count", n))
			}

			recs, err := repo.ClaimPending(ctx, cfg.OutboxBatchSize)
			if err != nil {
				m.claimErrorsTotal.Inc()
				log.Error("outbox_claim_failed", slog.String("err", err.Error()))
				continue
			}
			if len(recs) == 0 {
				m.lagSeconds.Set(0)
				continue
			}

			m.claimedTotal.Add(float64(len(recs)))
			m.lagSeconds.Set(time.Since(recs[0].CreatedAt).Seconds())

			for _, rec := range recs {
				log.Info("outbox_event",
					slog.Int64("id", rec.ID),
					slog.String("aggregate", rec.Aggregate),
					slog.String("aggregate_id", rec.AggregateID),
					slog.String("event_type", rec.EventType),
					slog.Int("attempts", rec.Attempts),
				)

				if err := repo.MarkSent(ctx, rec.ID); err != nil {
					m.markErrorsTotal.Inc()
					log.Error("outbox_mark_sent_failed", slog.Int64("id", rec.ID), slog.String("err", err.Error()))
					_ = repo.MarkPending(ctx, rec.ID)
					continue
				}
				m.sentTotal.Inc()
			}
		}
	}
}

type relayMetrics struct {
	pollsTotal         prometheus.Counter
	claimedTotal       prometheus.Counter
	sentTotal          prometheus.Counter
	claimErrorsTotal   prometheus.Counter
	markErrorsTotal    prometheus.Counter
	requeuedTotal      prometheus.Counter
	requeueErrorsTotal prometheus.Counter
	lagSeconds         prometheus.Gauge
}

func newRelayMetrics(reg prometheus.Registerer) *relayMetrics {
	m := &relayMetrics{
		pollsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "outbox_relay_polls_total",
			Help: "Total number of outbox polling ticks.",
		}),
		claimedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "outbox_relay_claimed_total",
			Help: "Total number of claimed outbox rows.",
		}),
		sentTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "outbox_relay_sent_total",
			Help: "Total number of outbox rows marked as sent.",
		}),
		claimErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "outbox_relay_claim_errors_total",
			Help: "Total number of claim errors.",
		}),
		markErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "outbox_relay_mark_errors_total",
			Help: "Total number of errors while updating outbox rows.",
		}),
		requeuedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "outbox_relay_requeued_total",
			Help: "Total number of stuck outbox rows requeued back to pending.",
		}),
		requeueErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "outbox_relay_requeue_errors_total",
			Help: "Total number of requeue errors.",
		}),
		lagSeconds: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "outbox_lag_seconds",
			Help: "Lag in seconds between now and the oldest claimed outbox row.",
		}),
	}

	reg.MustRegister(
		m.pollsTotal,
		m.claimedTotal,
		m.sentTotal,
		m.claimErrorsTotal,
		m.markErrorsTotal,
		m.requeuedTotal,
		m.requeueErrorsTotal,
		m.lagSeconds,
	)

	return m
}
