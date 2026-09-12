package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port               string
	DBPath             string
	PollInterval       time.Duration
	FootballDataAPIKey string
	DevMocks           bool
}

func Load() (Config, error) {
	cfg := Config{}

	port := envOr("PORT", "8080")
	cfg.Port = port

	dbPath := envOr("DB_PATH", "football.db")
	cfg.DBPath = dbPath

	seconds := envOr("POLL_INTERVAL_SECONDS", "15")
	n, err := strconv.Atoi(seconds)
	if err != nil {
		return Config{}, fmt.Errorf("POLL_INTERVAL_SECONDS: %w", err)
	}
	if n <= 0 {
		return Config{}, fmt.Errorf("POLL_INTERVAL_SECONDS must be positive, got %d", n)
	}
	cfg.PollInterval = time.Duration(n) * time.Second

	cfg.FootballDataAPIKey = os.Getenv("FOOTBALL_DATA_API_KEY")

	switch v := strings.ToLower(os.Getenv("DEV_MOCKS")); v {
	case "":
		cfg.DevMocks = false
	case "1", "true":
		cfg.DevMocks = true
	default:
		return Config{}, fmt.Errorf("DEV_MOCKS must be \"1\" or \"true\", got %q", v)
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
