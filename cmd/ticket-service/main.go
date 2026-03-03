package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/k1networth/servicedesk-lite/internal/shared/config"
	"github.com/k1networth/servicedesk-lite/internal/shared/db"
	"github.com/k1networth/servicedesk-lite/internal/shared/env"
	"github.com/k1networth/servicedesk-lite/internal/shared/httpx"
	"github.com/k1networth/servicedesk-lite/internal/shared/logger"
	"github.com/k1networth/servicedesk-lite/internal/ticket"
)

const appName = "ticket-service"

func main() {
	cfg := config.Load()
	log := logger.New(appName, cfg.AppEnv)

	ctx := context.Background()

	var store ticket.Store
	var readyz func(context.Context) error
	if cfg.DatabaseURL != "" {
		pg, err := db.OpenPostgres(ctx, db.PostgresConfig{
			DatabaseURL:  cfg.DatabaseURL,
			MaxOpenConns: env.Int("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: env.Int("DB_MAX_IDLE_CONNS", 25),
		})
		if err != nil {
			log.Error("db_open_failed", slog.String("err", err.Error()))
			return
		}
		defer func() {
			if err := pg.Close(); err != nil {
				log.Error("db_close_failed", slog.String("err", err.Error()))
			}
		}()

		store = ticket.NewPostgresStore(pg)
		readyz = func(ctx context.Context) error { return pg.PingContext(ctx) }
		log.Info("storage", slog.String("type", "postgres"))
	} else {
		store = ticket.NewInMemoryStore()
		log.Info("storage", slog.String("type", "memory"))
	}

	ticketH := &ticket.Handler{
		Log:   log,
		Store: store,
	}

	handler := httpx.NewRouter(log, ticketH, readyz)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Info("http_listen", slog.String("addr", srv.Addr))

	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Error("http_server_error", slog.String("err", err.Error()))
		}
	}()

	httpx.WaitAndShutdown(log, srv, 10*time.Second)
}
