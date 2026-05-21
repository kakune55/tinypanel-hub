package main

import (
	"context"
	"errors"
	"flag"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tinypanel-hub"
	"tinypanel-hub/internal/app"
	"tinypanel-hub/internal/config"
)

func main() {
	configPath := flag.String("config", config.PathFromEnv(), "path to JSON config file")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := ensureEtc(logger); err != nil {
		logger.Error("ensure etc assets", "err", err)
		os.Exit(1)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("load config", "path", *configPath, "err", err)
		os.Exit(1)
	}

	handler, err := app.NewHandler(cfg, logger)
	if err != nil {
		logger.Error("initialize app", "err", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		logger.Info("tinypanel hub listening", "addr", cfg.Server.Addr, "data", cfg.Storage.DataFile, "telemetry", cfg.Storage.TelemetryFile, "config", *configPath)
		errc <- server.ListenAndServe()
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigc:
		logger.Info("shutdown signal received", "signal", sig.String())
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped", "err", err)
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}

func ensureEtc(logger *slog.Logger) error {
	const etcDir = "etc"
	const examplePath = "etc/config.example.json"

	if err := os.MkdirAll(etcDir, 0755); err != nil {
		return err
	}

	if _, err := os.Stat(examplePath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	data, err := fs.ReadFile(tinypanelhub.EtcFS, examplePath)
	if err != nil {
		return err
	}

	if err := os.WriteFile(examplePath, data, 0644); err != nil {
		return err
	}
	logger.Info("created example config", "path", examplePath)
	return nil
}
