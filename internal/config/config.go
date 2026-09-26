package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultHTTPAddr        = ":8080"
	DefaultLogLevel        = "info"
	DefaultShutdownTimeout = 10 * time.Second
)

type Config struct {
	HTTPAddr        string
	LogLevel        string
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        envOrDefault("AIDI_HTTP_ADDR", DefaultHTTPAddr),
		LogLevel:        strings.ToLower(envOrDefault("AIDI_LOG_LEVEL", DefaultLogLevel)),
		ShutdownTimeout: DefaultShutdownTimeout,
	}

	if raw := os.Getenv("AIDI_SHUTDOWN_TIMEOUT"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("AIDI_SHUTDOWN_TIMEOUT: %w", err)
		}
		if d <= 0 {
			return Config{}, fmt.Errorf("AIDI_SHUTDOWN_TIMEOUT must be > 0")
		}
		cfg.ShutdownTimeout = d
	}

	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("AIDI_LOG_LEVEL must be one of debug, info, warn, error")
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func BoolEnv(key string, fallback bool) (bool, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	return value, nil
}
