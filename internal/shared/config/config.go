package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv      string
	HTTPAddr    string
	MetricsAddr string
	DatabaseURL string

	OutboxBatchSize         int
	OutboxPollInterval      time.Duration
	OutboxProcessingTimeout time.Duration
}

func Load() Config {
	loadDotEnv(".env")

	cfg := Config{
		AppEnv:      "dev",
		HTTPAddr:    ":8080",
		MetricsAddr: ":9090",

		OutboxBatchSize:         50,
		OutboxPollInterval:      500 * time.Millisecond,
		OutboxProcessingTimeout: 30 * time.Second,
	}

	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.AppEnv = v
	}
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	if v := os.Getenv("METRICS_ADDR"); v != "" {
		cfg.MetricsAddr = v
	}

	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}

	if v := os.Getenv("OUTBOX_BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.OutboxBatchSize = n
		}
	}
	if v := os.Getenv("OUTBOX_POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.OutboxPollInterval = d
		}
	}
	if v := os.Getenv("OUTBOX_PROCESSING_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.OutboxProcessingTimeout = d
		}
	}

	return cfg
}
