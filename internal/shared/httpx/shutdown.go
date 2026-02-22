package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func WaitAndShutdown(log *slog.Logger, srv *http.Server, timeout time.Duration) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	<-ctx.Done()
	stop()

	log.Info("shutdown_start")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_ = srv.Shutdown(shutdownCtx)

	log.Info("shutdown_done")
}
