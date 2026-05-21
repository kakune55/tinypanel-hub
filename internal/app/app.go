package app

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"tinypanel-hub/internal/config"
	"tinypanel-hub/internal/httpapi"
	"tinypanel-hub/internal/store"
	weatherapi "tinypanel-hub/internal/weather"
	"tinypanel-hub/internal/webui"
)

func NewHandler(cfg config.Config, logger *slog.Logger) (http.Handler, error) {
	dataStore, err := store.OpenFiles(cfg.Storage.DataFile, cfg.Storage.TelemetryFile)
	if err != nil {
		return nil, err
	}

	weatherProvider, err := NewWeatherProvider(cfg)
	if err != nil {
		return nil, err
	}

	return httpapi.New(dataStore, logger, httpapi.Options{
		APIToken:        cfg.Server.APIToken,
		WeatherProvider: weatherProvider,
		WebHandler:      webui.Handler(),
	}), nil
}

func NewWeatherProvider(cfg config.Config) (httpapi.WeatherProvider, error) {
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
