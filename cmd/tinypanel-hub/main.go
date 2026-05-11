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
	"strings"
	"syscall"
	"time"

	"tinypanel-hub"
	"tinypanel-hub/internal/config"
	"tinypanel-hub/internal/httpapi"
	"tinypanel-hub/internal/store"
	weatherapi "tinypanel-hub/internal/weather"
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

	dataStore, err := store.OpenFiles(cfg.Storage.DataFile, cfg.Storage.TelemetryFile)
	if err != nil {
		logger.Error("open store", "err", err)
		os.Exit(1)
	}

	weatherProvider, err := newWeatherProvider(cfg)
	if err != nil {
		logger.Error("configure weather provider", "err", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           httpapi.New(dataStore, logger, httpapi.Options{APIToken: cfg.Server.APIToken, WeatherProvider: weatherProvider}),
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

func newWeatherProvider(cfg config.Config) (httpapi.WeatherProvider, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Weather.Provider))
	if provider == "" || provider == "manual" || provider == "none" {
		return nil, nil
	}
	if provider != "qweather" {
		return nil, errors.New("unsupported weather provider " + cfg.Weather.Provider)
	}

	timeout, err := time.ParseDuration(cfg.Weather.Timeout)
	if err != nil {
		return nil, err
	}
	ttl, err := time.ParseDuration(cfg.Weather.CacheTTL)
	if err != nil {
		return nil, err
	}

	client, err := weatherapi.NewQWeatherClient(weatherapi.QWeatherOptions{
		APIHost:     cfg.Weather.APIHost,
		APIKey:      cfg.Weather.APIKey,
		BearerToken: cfg.Weather.BearerToken,
		Location:    cfg.Weather.Location,
		Lang:        cfg.Weather.Lang,
		Unit:        cfg.Weather.Unit,
		Hours:       cfg.Weather.Hours,
		Days:        cfg.Weather.Days,
		HTTPClient:  &http.Client{Timeout: timeout},
	})
	if err != nil {
		return nil, err
	}
	return weatherapi.NewCache(client, ttl), nil
}
