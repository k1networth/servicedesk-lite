package config

import "github.com/k1networth/servicedesk-lite/internal/shared/env"

type Config struct {
	AppEnv      string
	HTTPAddr    string
	DatabaseURL string
}

func Load() Config {
	loadDotEnv(".env")

	return Config{
		AppEnv:      env.String("APP_ENV", "dev"),
		HTTPAddr:    env.String("HTTP_ADDR", ":8080"),
		DatabaseURL: env.String("DATABASE_URL", ""),
	}
}
