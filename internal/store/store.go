package store

import (
	"errors"
	"sync"
	"time"

	"tinypanel-hub/internal/domain"
)

const (
	maxMessages  = 100
	maxTelemetry = 500
)

var errEmptyStorePath = errors.New("store paths must not be empty")

type FileStore struct {
	mu              sync.RWMutex
	state           *stateFile
	telemetry       *telemetryLog
	nextTelemetryID int64
}

func OpenFiles(statePath, telemetryPath string) (*FileStore, error) {
	if statePath == "" || telemetryPath == "" {
		return nil, errEmptyStorePath
	}

	s := &FileStore{
		state:     newStateFile(statePath),
		telemetry: newTelemetryLog(telemetryPath),
	}
	if err := s.state.load(); err != nil {
		return nil, err
	}

	items, err := s.telemetry.loadRecent(maxTelemetry)
	if err != nil {
		return nil, err
	}
	s.nextTelemetryID = nextTelemetryID(items)
	return s, nil
}

func defaultWeather() domain.Weather {
	return domain.Weather{
		Location:    "unknown",
		Condition:   "unknown",
		Temperature: 0,
		Humidity:    0,
		UpdatedAt:   time.Now().UTC(),
	}
}

func nextTelemetryID(items []domain.Telemetry) int64 {
	next := int64(1)
	for _, item := range items {
		if item.ID >= next {
			next = item.ID + 1
		}
	}
	return next
}
