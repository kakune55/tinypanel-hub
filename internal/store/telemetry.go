package store

import (
	"time"

	"tinypanel-hub/internal/domain"
)

func (s *FileStore) Telemetry(limit int) []domain.Telemetry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, err := s.telemetry.loadRecent(limit)
	if err != nil {
		return nil
	}
	reverseTelemetry(items)
	return items
}

func (s *FileStore) AddTelemetry(t domain.Telemetry) (domain.Telemetry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	t.ID = s.nextTelemetryID
	t.ReceivedAt = now
	if t.ReportTimestamp.IsZero() {
		t.ReportTimestamp = now
	}
	if t.App == nil {
		t.App = map[string]any{}
	}

	if err := s.telemetry.append(t); err != nil {
		return domain.Telemetry{}, err
	}
	s.nextTelemetryID++
	return t, nil
}

func reverseTelemetry(items []domain.Telemetry) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}
