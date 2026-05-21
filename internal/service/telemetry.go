package service

import "tinypanel-hub/internal/domain"

type TelemetryService struct {
	store TelemetryStore
}

func (s TelemetryService) List(limit int) []domain.Telemetry {
	return s.store.Telemetry(limit)
}

func (s TelemetryService) DeviceList(ownerID, deviceID string, limit int) []domain.Telemetry {
	return s.store.DeviceTelemetry(ownerID, deviceID, limit)
}

func (s TelemetryService) Create(item domain.Telemetry) (domain.Telemetry, error) {
	return s.store.AddTelemetry(item)
}

func (s TelemetryService) CreateBatch(items []domain.Telemetry) ([]domain.Telemetry, error) {
	out := make([]domain.Telemetry, 0, len(items))
	for _, item := range items {
		stored, err := s.store.AddTelemetry(item)
		if err != nil {
			return nil, err
		}
		out = append(out, stored)
	}
	return out, nil
}
