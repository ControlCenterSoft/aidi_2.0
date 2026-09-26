package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("AIDI_HTTP_ADDR", "")
	t.Setenv("AIDI_LOG_LEVEL", "")
	t.Setenv("AIDI_SHUTDOWN_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != DefaultHTTPAddr {
		t.Fatalf("expected %q, got %q", DefaultHTTPAddr, cfg.HTTPAddr)
	}
	if cfg.LogLevel != DefaultLogLevel {
		t.Fatalf("expected %q, got %q", DefaultLogLevel, cfg.LogLevel)
	}
	if cfg.ShutdownTimeout != DefaultShutdownTimeout {
		t.Fatalf("expected %s, got %s", DefaultShutdownTimeout, cfg.ShutdownTimeout)
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	t.Setenv("AIDI_LOG_LEVEL", "verbose")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid log level error")
	}
}

func TestLoadRejectsInvalidShutdownTimeout(t *testing.T) {
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
}
