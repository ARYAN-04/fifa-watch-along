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

	port, err := envOr("PORT", "8080")
	if err != nil {
		return Config{}, err
	}
	cfg.Port = port

	dbPath, err := envOr("DB_PATH", "football.db")
	if err != nil {
		return Config{}, err
	}
	cfg.DBPath = dbPath

	seconds, err := envOr("POLL_INTERVAL_SECONDS", "15")
	if err != nil {
		return Config{}, err
	}
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

func envOr(key, fallback string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	return v, nil
}
