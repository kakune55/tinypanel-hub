package service

import (
	"context"

	"tinypanel-hub/internal/domain"
)

type SnapshotService struct {
	store   SnapshotStore
	weather WeatherProvider
}

func (s SnapshotService) Get(ctx context.Context) (domain.Snapshot, error) {
	snapshot := s.store.Snapshot()
	if s.weather != nil {
		weather, err := s.weather.Current(ctx)
		if err != nil {
			return domain.Snapshot{}, err
		}
		snapshot.Weather = weather
	}
	return snapshot, nil
}
