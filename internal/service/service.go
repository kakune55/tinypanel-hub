package service

import (
	"context"

	"tinypanel-hub/internal/domain"
)

type WeatherProvider interface {
	Current(context.Context) (domain.Weather, error)
}

type SnapshotStore interface {
	Snapshot() domain.Snapshot
}

type WeatherStore interface {
	Weather() domain.Weather
}

type MessageStore interface {
	Messages(limit int) []domain.Message
	Message(id int64) (domain.Message, bool)
	AddMessage(channel, author, body string) (domain.Message, error)
	MessageSubscription(deviceID, channel string, limit int) domain.MessageSubscription
	AckMessage(deviceID string, messageID int64) (bool, error)
}

type TodoStore interface {
	Todos() []domain.Todo
	Todo(id int64) (domain.Todo, bool)
	AddTodo(text string, status int) (domain.Todo, error)
	UpdateTodo(id, version int64, patch domain.TodoPatch) (domain.Todo, bool, bool, error)
	DeleteTodo(id, version int64) (bool, bool, error)
}

type TelemetryStore interface {
	Telemetry(limit int) []domain.Telemetry
	AddTelemetry(domain.Telemetry) (domain.Telemetry, error)
}

type Store interface {
	SnapshotStore
	WeatherStore
	MessageStore
	TodoStore
	TelemetryStore
}

type Services struct {
	Snapshot  SnapshotService
	Weather   WeatherService
	Messages  MessageService
	Todos     TodoService
	Telemetry TelemetryService
}

func New(store Store, weather WeatherProvider) Services {
	return Services{
		Snapshot:  SnapshotService{store: store, weather: weather},
		Weather:   WeatherService{store: store, weather: weather},
		Messages:  MessageService{store: store},
		Todos:     TodoService{store: store},
		Telemetry: TelemetryService{store: store},
	}
}
