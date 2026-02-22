package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/k1networth/servicedesk-lite/internal/shared/config"
	"github.com/k1networth/servicedesk-lite/internal/shared/httpx"
	"github.com/k1networth/servicedesk-lite/internal/shared/logger"
	"github.com/k1networth/servicedesk-lite/internal/ticket"
)

const appName = "ticket-service"

func main() {
	cfg := config.Load()
	log := logger.New(appName, cfg.AppEnv)

	store := ticket.NewInMemoryStore()
	ticketH := &ticket.Handler{
		Log:   log,
		Store: store,
	}

	handler := httpx.NewRouter(log, ticketH)

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
