package service

import (
	"context"

	"tinypanel-hub/internal/domain"
)

type WeatherService struct {
	store   WeatherStore
	weather WeatherProvider
}

func (s WeatherService) Get(ctx context.Context) (domain.Weather, error) {
	if s.weather != nil {
		return s.weather.Current(ctx)
	}
	return s.store.Weather(), nil
}
