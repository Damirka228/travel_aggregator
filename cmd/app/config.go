package main

import (
	"fmt"
	"os"
	"strconv"
)

type config struct {
	port               string
	databaseURL        string
	redisAddr          string
	travelpayoutsToken string
	enableProfiler     bool
}

func loadConfig() (config, error) {
	cfg := config{
		port:               envOrDefault("PORT", "8080"),
		databaseURL:        envOrDefault("DATABASE_URL", os.Getenv("POSTGRESQL_STR")),
		redisAddr:          envOrDefault("REDIS_ADDR", envOrDefault("REDIS_URL", "localhost:6379")),
		travelpayoutsToken: os.Getenv("TRAVELPAYOUTS_TOKEN"),
	}

	port, err := strconv.Atoi(cfg.port)
	if err != nil || port < 1 || port > 65535 {
		return config{}, fmt.Errorf("PORT must be an integer between 1 and 65535")
	}
	if cfg.databaseURL == "" {
		return config{}, fmt.Errorf("DATABASE_URL is required (POSTGRESQL_STR is also supported)")
	}
	if cfg.travelpayoutsToken == "" {
		return config{}, fmt.Errorf("TRAVELPAYOUTS_TOKEN is required")
	}
	if os.Getenv("JWT_SECRET") == "" {
		return config{}, fmt.Errorf("JWT_SECRET is required")
	}

	cfg.enableProfiler, err = strconv.ParseBool(envOrDefault("ENABLE_PPROF", "false"))
	if err != nil {
		return config{}, fmt.Errorf("ENABLE_PPROF must be a boolean: %w", err)
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
