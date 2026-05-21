package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tinypanel-hub/internal/domain"
)

func TestTelemetryUsesJSONL(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	telemetryPath := filepath.Join(dir, "telemetry.jsonl")

	s, err := OpenFiles(statePath, telemetryPath)
	if err != nil {
		t.Fatal(err)
	}

	first := telemetryFixture(1)
	second := telemetryFixture(2)
	if _, err := s.AddTelemetry(first); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddTelemetry(second); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(telemetryPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 {
		t.Fatalf("jsonl lines = %d, want 2; content=%s", len(lines), b)
	}

	var stored domain.Telemetry
	if err := json.Unmarshal([]byte(lines[0]), &stored); err != nil {
		t.Fatal(err)
	}
	if stored.ID != 1 || stored.Sequence != 1 {
		t.Fatalf("first telemetry = %+v", stored)
	}

	got := s.Telemetry(2)
	if len(got) != 2 || got[0].ID != 2 || got[1].ID != 1 {
		t.Fatalf("telemetry order = %+v", got)
	}
}

func TestMessagesStayInStateFile(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	telemetryPath := filepath.Join(dir, "telemetry.jsonl")

	s, err := OpenFiles(statePath, telemetryPath)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := s.AddMessage("desk", "hub", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if msg.ID != 1 {
		t.Fatalf("message id = %d, want 1", msg.ID)
	}
	if ok, err := s.AckMessage("tinypanel-001", msg.ID); err != nil || !ok {
		t.Fatalf("ack ok=%v err=%v", ok, err)
	}

	reopened, err := OpenFiles(statePath, telemetryPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reopened.Message(msg.ID); !ok {
		t.Fatalf("message not persisted")
	}
	sub := reopened.MessageSubscription("tinypanel-001", "desk", 10)
	if sub.UnreadCount != 0 {
		t.Fatalf("subscription after ack = %+v", sub)
	}
}

func TestMessageAcksPrunedWhenMessagesRotate(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	telemetryPath := filepath.Join(dir, "telemetry.jsonl")

	s, err := OpenFiles(statePath, telemetryPath)
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.AddMessage("desk", "hub", "old")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := s.AckMessage("tinypanel-001", first.ID); err != nil || !ok {
		t.Fatalf("ack ok=%v err=%v", ok, err)
	}
	for i := 0; i < maxMessages; i++ {
		if _, err := s.AddMessage("desk", "hub", "new"); err != nil {
			t.Fatal(err)
		}
	}

	reopened, err := OpenFiles(statePath, telemetryPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reopened.Message(first.ID); ok {
		t.Fatalf("first message should have rotated out")
	}
	if ids := reopened.state.data.MessageAcks["tinypanel-001"]; len(ids) != 0 {
		t.Fatalf("stale ack ids = %v", ids)
	}
}

func TestTodosPersistAndUseCAS(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	telemetryPath := filepath.Join(dir, "telemetry.jsonl")

	s, err := OpenFiles(statePath, telemetryPath)
	if err != nil {
		t.Fatal(err)
	}
	todo, err := s.AddTodo("写 TODO 功能", 0)
	if err != nil {
		t.Fatal(err)
	}
	if todo.ID != 1 || todo.Version != 1 {
		t.Fatalf("created todo = %+v", todo)
	}

	status := 1
	updated, found, swapped, err := s.UpdateTodo(todo.ID, todo.Version, domain.TodoPatch{Status: &status})
	if err != nil {
		t.Fatal(err)
	}
	if !found || !swapped || updated.Status != 1 || updated.Version != 2 {
		t.Fatalf("updated=%+v found=%v swapped=%v", updated, found, swapped)
	}

	status = 2
	_, found, swapped, err = s.UpdateTodo(todo.ID, 1, domain.TodoPatch{Status: &status})
	if err != nil {
		t.Fatal(err)
	}
	if !found || swapped {
		t.Fatalf("stale update found=%v swapped=%v", found, swapped)
	}

	reopened, err := OpenFiles(statePath, telemetryPath)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopened.Todo(todo.ID)
	if !ok || got.Status != 1 || got.Version != 2 {
		t.Fatalf("reopened todo = %+v ok=%v", got, ok)
	}

	found, swapped, err = reopened.DeleteTodo(todo.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !found || swapped {
		t.Fatalf("stale delete found=%v swapped=%v", found, swapped)
	}

	found, swapped, err = reopened.DeleteTodo(todo.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !found || !swapped {
		t.Fatalf("delete found=%v swapped=%v", found, swapped)
	}
}

func telemetryFixture(sequence int64) domain.Telemetry {
	return domain.Telemetry{
		SchemaVersion:   1,
		DeviceID:        "tinypanel-001",
		BootID:          "boot",
		Sequence:        sequence,
		ReportTimestamp: time.Date(2026, 5, 10, 16, 20, 0, 0, time.UTC),
		App:             map[string]any{},
	}
}
