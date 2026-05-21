package httpapi

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tinypanel-hub/internal/domain"
)

type fixedWeatherProvider struct {
	weather domain.Weather
}

func (p fixedWeatherProvider) Current(context.Context) (domain.Weather, error) {
	return p.weather, nil
}

type memoryStore struct {
	weather   domain.Weather
	messages  []domain.Message
	acks      map[string]map[int64]bool
	todos     []domain.Todo
	telemetry []domain.Telemetry
}

func (s *memoryStore) Snapshot() domain.Snapshot {
	return domain.Snapshot{
		Weather:   s.weather,
		Messages:  s.messages,
		Todos:     s.todos,
		Telemetry: s.telemetry,
	}
}

func (s *memoryStore) Weather() domain.Weather {
	return s.weather
}

func (s *memoryStore) Messages(limit int) []domain.Message {
	return s.messages
}

func (s *memoryStore) Message(id int64) (domain.Message, bool) {
	for _, msg := range s.messages {
		if msg.ID == id {
			return msg, true
		}
	}
	return domain.Message{}, false
}

func (s *memoryStore) AddMessage(channel, author, body string) (domain.Message, error) {
	msg := domain.Message{ID: int64(len(s.messages) + 1), Channel: channel, Author: author, Body: body, CreatedAt: time.Now().UTC()}
	s.messages = append(s.messages, msg)
	return msg, nil
}

func (s *memoryStore) MessageSubscription(deviceID, channel string, limit int) domain.MessageSubscription {
	sub := domain.MessageSubscription{DeviceID: deviceID, Channel: channel}
	for _, msg := range s.messages {
		if msg.Channel != channel || s.acks[deviceID][msg.ID] {
			continue
		}
		sub.UnreadCount++
		if limit <= 0 || len(sub.MessageIDs) < limit {
			sub.MessageIDs = append(sub.MessageIDs, msg.ID)
		}
	}
	return sub
}

func (s *memoryStore) AckMessage(deviceID string, messageID int64) (bool, error) {
	if _, ok := s.Message(messageID); !ok {
		return false, nil
	}
	if s.acks == nil {
		s.acks = map[string]map[int64]bool{}
	}
	if s.acks[deviceID] == nil {
		s.acks[deviceID] = map[int64]bool{}
	}
	s.acks[deviceID][messageID] = true
	return true, nil
}

func (s *memoryStore) Todos() []domain.Todo {
	return s.todos
}

func (s *memoryStore) Todo(id int64) (domain.Todo, bool) {
	for _, todo := range s.todos {
		if todo.ID == id {
			return todo, true
		}
	}
	return domain.Todo{}, false
}

func (s *memoryStore) AddTodo(text string, status int) (domain.Todo, error) {
	now := time.Now().UTC()
	todo := domain.Todo{ID: int64(len(s.todos) + 1), Text: text, Status: status, Version: 1, CreatedAt: now, UpdatedAt: now}
	s.todos = append(s.todos, todo)
	return todo, nil
}

func (s *memoryStore) UpdateTodo(id, version int64, patch domain.TodoPatch) (domain.Todo, bool, bool, error) {
	for i, todo := range s.todos {
		if todo.ID != id {
			continue
		}
		if todo.Version != version {
			return todo, true, false, nil
		}
		if patch.Text != nil {
			todo.Text = *patch.Text
		}
		if patch.Status != nil {
			todo.Status = *patch.Status
		}
		todo.Version++
		todo.UpdatedAt = time.Now().UTC()
		s.todos[i] = todo
		return todo, true, true, nil
	}
	return domain.Todo{}, false, false, nil
}

func (s *memoryStore) DeleteTodo(id, version int64) (bool, bool, error) {
	for i, todo := range s.todos {
		if todo.ID != id {
			continue
		}
		if todo.Version != version {
			return true, false, nil
		}
		s.todos = append(s.todos[:i], s.todos[i+1:]...)
		return true, true, nil
	}
	return false, false, nil
}

func (s *memoryStore) Telemetry(limit int) []domain.Telemetry {
	return s.telemetry
}

func (s *memoryStore) AddTelemetry(t domain.Telemetry) (domain.Telemetry, error) {
	t.ID = int64(len(s.telemetry) + 1)
	t.ReceivedAt = time.Now().UTC()
	s.telemetry = append(s.telemetry, t)
	return t, nil
}

func TestAPITokenRequired(t *testing.T) {
	handler := New(newMemoryStore(), slog.Default(), Options{APIToken: "secret"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestTelemetryUpload(t *testing.T) {
	store := newMemoryStore()
	handler := New(store, slog.Default(), Options{APIToken: "secret"})

	body := []byte(`{
		"schema_version": 1,
		"device_id": "tinypanel-001",
		"boot_id": "boot_20260510_162000",
		"sequence": 123,
		"report_timestamp": "2026-05-10T16:20:00+08:00",
		"uptime_s": 8642300,
		"power": {
			"battery": {
				"raw_adc": 1882,
				"raw_voltage_mv": 3950,
				"voltage_mv": 3950,
				"percentage": 78,
				"status": "discharging"
			},
			"usb_connected": false
		},
		"environment": {
			"shtc3": {
				"temperature_c": 22.5,
				"humidity_rh": 45.0,
				"sensor_ok": true
			}
		},
		"network": {
			"wifi_connected": true,
			"ssid": "HomeNetwork",
			"rssi_dbm": -60,
			"ip": "192.168.3.42"
		},
		"system": {
			"free_heap_bytes": 184320,
			"free_psram_bytes": 6291456,
			"ntp_sync": true
		},
		"storage": {
			"sd_card_present": true,
			"sd_card_total_mb": 2048,
			"sd_card_used_mb": 512
		},
		"app": {}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if len(store.telemetry) != 1 {
		t.Fatalf("telemetry count = %d, want 1", len(store.telemetry))
	}
	item := store.telemetry[0]
	if item.DeviceID != "tinypanel-001" || item.Environment.SHTC3.TemperatureC != 22.5 {
		t.Fatalf("stored telemetry mismatch: %+v", item)
	}
}

func TestTelemetryBatchUpload(t *testing.T) {
	store := newMemoryStore()
	handler := New(store, slog.Default(), Options{APIToken: "secret"})

	body := `{"items":[
		{
			"schema_version": 1,
			"device_id": "tinypanel-001",
			"boot_id": "boot",
			"sequence": 1,
			"report_timestamp": "2026-05-10T16:20:00+08:00",
			"power": {"battery": {"status": "discharging"}},
			"environment": {"shtc3": {}},
			"network": {},
			"system": {},
			"storage": {},
			"app": {}
		},
		{
			"schema_version": 1,
			"device_id": "tinypanel-001",
			"boot_id": "boot",
			"sequence": 2,
			"report_timestamp": "2026-05-10T16:21:00+08:00",
			"power": {"battery": {"status": "discharging"}},
			"environment": {"shtc3": {}},
			"network": {},
			"system": {},
			"storage": {},
			"app": {}
		}
	]}`

	rec := postJSON(t, handler, "/api/v1/telemetry/batch", body, http.StatusCreated)
	if len(store.telemetry) != 2 {
		t.Fatalf("telemetry count = %d, want 2", len(store.telemetry))
	}
	if !strings.Contains(rec.Body.String(), `"count":2`) {
		t.Fatalf("unexpected telemetry batch response: %s", rec.Body.String())
	}
}

func TestMessageSubscriptionFlow(t *testing.T) {
	store := newMemoryStore()
	handler := New(store, slog.Default(), Options{APIToken: "secret"})

	postJSON(t, handler, "/api/v1/messages", `{"channel":"desk","author":"hub","body":"hello"}`, http.StatusCreated)

	rec := getJSON(t, handler, "/api/v1/subscriptions/desk?device_id=tinypanel-001", http.StatusOK)
	if !strings.Contains(rec.Body.String(), `"unread_count":1`) || !strings.Contains(rec.Body.String(), `"message_ids":[1]`) {
		t.Fatalf("unexpected subscription response: %s", rec.Body.String())
	}

	getJSON(t, handler, "/api/v1/messages/1", http.StatusOK)
	postJSON(t, handler, "/api/v1/messages/1/ack", `{"device_id":"tinypanel-001"}`, http.StatusOK)

	rec = getJSON(t, handler, "/api/v1/subscriptions/desk?device_id=tinypanel-001", http.StatusOK)
	if !strings.Contains(rec.Body.String(), `"unread_count":0`) {
		t.Fatalf("unexpected subscription response after ack: %s", rec.Body.String())
	}
}

func TestMessageBatchAckUsesDeviceHeader(t *testing.T) {
	store := newMemoryStore()
	handler := New(store, slog.Default(), Options{APIToken: "secret"})

	postJSON(t, handler, "/api/v1/messages", `{"channel":"desk","author":"hub","body":"first"}`, http.StatusCreated)
	postJSON(t, handler, "/api/v1/messages", `{"channel":"desk","author":"hub","body":"second"}`, http.StatusCreated)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/ack", strings.NewReader(`{"message_ids":[1,2]}`))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("X-Device-ID", "tinypanel-001")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"acked_ids":[1,2]`) {
		t.Fatalf("unexpected batch ack response: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/desk", nil)
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("X-Device-ID", "tinypanel-001")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"unread_count":0`) {
		t.Fatalf("unexpected subscription response: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSnapshotInclude(t *testing.T) {
	store := newMemoryStore()
	store.weather = domain.Weather{Location: "stored", Condition: "stored"}
	store.todos = []domain.Todo{{ID: 1, Text: "todo", Version: 1}}
	handler := New(store, slog.Default(), Options{APIToken: "secret"})

	rec := getJSON(t, handler, "/api/v1/snapshot?include=weather,todos", http.StatusOK)
	body := rec.Body.String()
	if !strings.Contains(body, `"weather":`) || !strings.Contains(body, `"todos":`) {
		t.Fatalf("filtered snapshot missing requested fields: %s", body)
	}
	if strings.Contains(body, `"messages":`) || strings.Contains(body, `"telemetry":`) {
		t.Fatalf("filtered snapshot contains unrequested fields: %s", body)
	}
}

func TestTodoCASFlow(t *testing.T) {
	store := newMemoryStore()
	handler := New(store, slog.Default(), Options{APIToken: "secret"})

	rec := postJSON(t, handler, "/api/v1/todos", `{"text":"写测试","status":0}`, http.StatusCreated)
	body := rec.Body.String()
	if !strings.Contains(body, `"id":1`) || !strings.Contains(body, `"version":1`) || !strings.Contains(body, `"status":0`) {
		t.Fatalf("unexpected todo create response: %s", body)
	}

	rec = patchJSON(t, handler, "/api/v1/todos/1", `{"version":1,"status":1}`, http.StatusOK)
	if !strings.Contains(rec.Body.String(), `"version":2`) || !strings.Contains(rec.Body.String(), `"status":1`) {
		t.Fatalf("unexpected todo patch response: %s", rec.Body.String())
	}

	patchJSON(t, handler, "/api/v1/todos/1", `{"version":1,"status":2}`, http.StatusConflict)
	deleteJSON(t, handler, "/api/v1/todos/1", `{"version":2}`, http.StatusNoContent)

	getJSON(t, handler, "/api/v1/todos/1", http.StatusNotFound)
}

func TestTodoValidation(t *testing.T) {
	store := newMemoryStore()
	handler := New(store, slog.Default(), Options{APIToken: "secret"})

	postJSON(t, handler, "/api/v1/todos", `{"text":"","status":0}`, http.StatusBadRequest)
	postJSON(t, handler, "/api/v1/todos", `{"text":"bad","status":3}`, http.StatusBadRequest)
	postJSON(t, handler, "/api/v1/todos", `{"text":"123456789012345678901234567890123456789012345678901","status":0}`, http.StatusBadRequest)
}

func TestGetWeatherUsesProvider(t *testing.T) {
	store := newMemoryStore()
	store.weather = domain.Weather{Location: "stored", Condition: "stored"}
	handler := New(store, slog.Default(), Options{
		APIToken: "secret",
		WeatherProvider: fixedWeatherProvider{weather: domain.Weather{
			Location:    "101020100",
			Condition:   "晴",
			Temperature: 27,
			Humidity:    56,
		}},
	})

	rec := getJSON(t, handler, "/api/v1/weather", http.StatusOK)
	body := rec.Body.String()
	if !strings.Contains(body, `"location":"101020100"`) || !strings.Contains(body, `"condition":"晴"`) {
		t.Fatalf("unexpected weather response: %s", body)
	}
}

func newMemoryStore() *memoryStore {
	return &memoryStore{acks: map[string]map[int64]bool{}}
}

func postJSON(t *testing.T, handler http.Handler, path, body string, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != wantStatus {
		t.Fatalf("%s status = %d, want %d; body=%s", path, rec.Code, wantStatus, rec.Body.String())
	}
	return rec
}

func patchJSON(t *testing.T, handler http.Handler, path, body string, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != wantStatus {
		t.Fatalf("%s status = %d, want %d; body=%s", path, rec.Code, wantStatus, rec.Body.String())
	}
	return rec
}

func deleteJSON(t *testing.T, handler http.Handler, path, body string, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodDelete, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != wantStatus {
		t.Fatalf("%s status = %d, want %d; body=%s", path, rec.Code, wantStatus, rec.Body.String())
	}
	return rec
}

func getJSON(t *testing.T, handler http.Handler, path string, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != wantStatus {
		t.Fatalf("%s status = %d, want %d; body=%s", path, rec.Code, wantStatus, rec.Body.String())
	}
	return rec
}
