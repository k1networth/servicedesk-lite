package logger

import (
	"log/slog"
	"os"
)

func New(app, env string) *slog.Logger {
	level := slog.LevelInfo
	if env == "dev" || env == "local" {
		level = slog.LevelDebug
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(h).With(
		slog.String("app", app),
		slog.String("env", env),
	)
}
