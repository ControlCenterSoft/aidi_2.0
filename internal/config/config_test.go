package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("AIDI_HTTP_ADDR", "")
	t.Setenv("AIDI_LOG_LEVEL", "")
	t.Setenv("AIDI_SHUTDOWN_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != "" {
		t.Fatalf("expected explicit empty HTTP addr to remain empty for validation, got %q", cfg.HTTPAddr)
	}
}

func TestLoadDeterministicDefaults(t *testing.T) {
	t.Setenv("AIDI_HTTP_ADDR", DefaultHTTPAddr)
	t.Setenv("AIDI_LOG_LEVEL", DefaultLogLevel)
	t.Setenv("AIDI_SHUTDOWN_TIMEOUT", DefaultShutdownTimeout.String())

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != DefaultHTTPAddr || cfg.LogLevel != DefaultLogLevel || cfg.ShutdownTimeout != DefaultShutdownTimeout {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	t.Setenv("AIDI_HTTP_ADDR", DefaultHTTPAddr)
	t.Setenv("AIDI_LOG_LEVEL", "verbose")
	t.Setenv("AIDI_SHUTDOWN_TIMEOUT", "10s")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid log level error")
	}
}

func TestLoadRejectsInvalidShutdownTimeout(t *testing.T) {
	t.Setenv("AIDI_HTTP_ADDR", DefaultHTTPAddr)
	t.Setenv("AIDI_LOG_LEVEL", DefaultLogLevel)
	t.Setenv("AIDI_SHUTDOWN_TIMEOUT", "0s")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid shutdown timeout error")
	}
}

func TestBoolEnv(t *testing.T) {
	t.Setenv("AIDI_TEST_BOOL", "true")
	got, err := BoolEnv("AIDI_TEST_BOOL", false)
	if err != nil || !got {
		t.Fatalf("expected true, got %v err=%v", got, err)
	}

	t.Setenv("AIDI_TEST_BOOL", "not-bool")
	if _, err := BoolEnv("AIDI_TEST_BOOL", false); err == nil {
		t.Fatal("expected parse error")
	}

	_ = time.Second
}
