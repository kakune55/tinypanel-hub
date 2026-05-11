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
