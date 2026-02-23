package config

import "os"

type Config struct {
	AppEnv       string
	HTTPAddr     string
	DatabaseURL  string
}

func Load() Config {
	loadDotEnv(".env")

	cfg := Config{
		AppEnv:   "dev",
		HTTPAddr: ":8080",
	}

	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.AppEnv = v
	}
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}

	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}

	return cfg
}
