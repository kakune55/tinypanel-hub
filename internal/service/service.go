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

type UserStore interface {
	UserByTokenHash(tokenHash string) (domain.User, bool)
	User(id string) (domain.User, bool)
	CreateUser(name, email, tokenHash string) (domain.User, error)
}

type DeviceStore interface {
	DeviceByCredentials(deviceID, secretHash string) (domain.Device, bool)
	Device(ownerID, deviceID string) (domain.Device, bool)
	Devices(ownerID string) []domain.Device
	HelloDevice(deviceID, secretHash string) (domain.DeviceHello, error)
	BindDevice(ownerID, bindCode, name string) (domain.Device, bool, bool, error)
	UpdateDevice(ownerID, deviceID, name string) (domain.Device, bool, error)
	DeleteDevice(ownerID, deviceID string) (bool, error)
}

type MessageStore interface {
	DeviceMessages(ownerID, deviceID string, limit int) []domain.Message
	AddDeviceMessage(ownerID, deviceID, authorID, body, priority string) (domain.Message, error)
	PendingDeviceMessages(deviceID string, limit int) []domain.Message
	AckDeviceMessages(deviceID string, messageIDs []int64) ([]int64, []int64, error)
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
	DeviceTelemetry(ownerID, deviceID string, limit int) []domain.Telemetry
	AddTelemetry(domain.Telemetry) (domain.Telemetry, error)
}

type Store interface {
	SnapshotStore
	WeatherStore
	UserStore
	DeviceStore
	MessageStore
	TodoStore
	TelemetryStore
}

type Services struct {
	Snapshot  SnapshotService
	Weather   WeatherService
	Users     UserService
	Devices   DeviceService
	Messages  MessageService
	Todos     TodoService
	Telemetry TelemetryService
}

func New(store Store, weather WeatherProvider) Services {
	return Services{
		Snapshot:  SnapshotService{store: store, weather: weather},
		Weather:   WeatherService{store: store, weather: weather},
		Users:     UserService{store: store},
		Devices:   DeviceService{store: store},
		Messages:  MessageService{store: store},
		Todos:     TodoService{store: store},
		Telemetry: TelemetryService{store: store},
	}
}
