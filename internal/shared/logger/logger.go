package logger

import (
	"log/slog"
	"os"
)

func New(app, env string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	return slog.New(h).With(
		slog.String("app", app),
		slog.String("env", env),
	)
}
