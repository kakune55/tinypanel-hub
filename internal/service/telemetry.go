package service

import "tinypanel-hub/internal/domain"

type TelemetryService struct {
	store TelemetryStore
}

func (s TelemetryService) List(limit int) []domain.Telemetry {
	return s.store.Telemetry(limit)
}

func (s TelemetryService) Create(item domain.Telemetry) (domain.Telemetry, error) {
	return s.store.AddTelemetry(item)
}
