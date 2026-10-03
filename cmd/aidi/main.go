package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ControlCenterSoft/aidi_2.0/internal/buildinfo"
	"github.com/ControlCenterSoft/aidi_2.0/internal/config"
	"github.com/ControlCenterSoft/aidi_2.0/internal/health"
)

var (
	version    = "dev"
	commit     = "unknown"
	buildTime  = "unknown"
	provenance = "unknown"
)

func main() {
	build := buildinfo.New(version, commit, buildTime, provenance)
	if buildinfo.VersionRequested(os.Args) {
		if err := buildinfo.WriteJSON(os.Stdout, build); err != nil {
			fmt.Fprintln(os.Stderr, "write build metadata:", err)
			os.Exit(1)
		}
		return
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(2)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel(cfg.LogLevel)}))
	slog.SetDefault(logger)

	var ready atomic.Bool

	mux := http.NewServeMux()
	mux.HandleFunc("/health", health.Live(build))
	mux.HandleFunc("/ready", health.Ready(build, ready.Load))

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("aidi core starting",
			"addr", cfg.HTTPAddr,
			"version", build.Version,
			"commit", build.Commit,
			"build_time", build.BuildTime,
			"provenance", build.Provenance,
		)
		ready.Store(true)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-errCh:
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}

	ready.Store(false)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("aidi core stopped")
}

func logLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
