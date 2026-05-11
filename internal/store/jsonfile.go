package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"tinypanel-hub/internal/domain"
)

const (
	maxMessages  = 100
	maxTelemetry = 500
)

type JSONFile struct {
	path string
	mu   sync.RWMutex
	data dataFile
}

type dataFile struct {
	NextMessageID   int64              `json:"next_message_id"`
	NextTelemetryID int64              `json:"next_telemetry_id"`
	Weather         domain.Weather     `json:"weather"`
	Messages        []domain.Message   `json:"messages"`
	MessageAcks     map[string][]int64 `json:"message_acks"`
	Telemetry       []domain.Telemetry `json:"telemetry"`
}

func OpenJSONFile(path string) (*JSONFile, error) {
	s := &JSONFile{
		path: path,
		data: dataFile{
			NextMessageID:   1,
			NextTelemetryID: 1,
			MessageAcks:     map[string][]int64{},
			Weather: domain.Weather{
				Location:    "unknown",
				Condition:   "unknown",
				Temperature: 0,
				Humidity:    0,
				UpdatedAt:   time.Now().UTC(),
			},
		},
	}

	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *JSONFile) Snapshot() domain.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return domain.Snapshot{
		Weather:   s.data.Weather,
		Messages:  append([]domain.Message(nil), s.data.Messages...),
		Telemetry: append([]domain.Telemetry(nil), s.data.Telemetry...),
	}
}

func (s *JSONFile) Weather() domain.Weather {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Weather
}

func (s *JSONFile) Messages(limit int) []domain.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.data.Messages
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}

	start := len(items) - limit
	out := append([]domain.Message(nil), items[start:]...)
	reverseMessages(out)
	return out
}

func (s *JSONFile) Message(id int64) (domain.Message, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, msg := range s.data.Messages {
		if msg.ID == id {
			return msg, true
		}
	}
	return domain.Message{}, false
}

func (s *JSONFile) AddMessage(channel, author, body string) (domain.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := domain.Message{
		ID:        s.data.NextMessageID,
		Channel:   channel,
		Author:    author,
		Body:      body,
		CreatedAt: time.Now().UTC(),
	}
	s.data.NextMessageID++
	s.data.Messages = append(s.data.Messages, msg)
	if len(s.data.Messages) > maxMessages {
		s.data.Messages = s.data.Messages[len(s.data.Messages)-maxMessages:]
	}
	return msg, s.saveLocked()
}

func (s *JSONFile) MessageSubscription(deviceID, channel string, limit int) domain.MessageSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	acked := make(map[int64]bool)
	for _, id := range s.data.MessageAcks[deviceID] {
		acked[id] = true
	}

	sub := domain.MessageSubscription{
		DeviceID:   deviceID,
		Channel:    channel,
		MessageIDs: []int64{},
	}

	for _, msg := range s.data.Messages {
		if msg.Channel != channel || acked[msg.ID] {
			continue
		}
		sub.UnreadCount++
		if limit <= 0 || len(sub.MessageIDs) < limit {
			sub.MessageIDs = append(sub.MessageIDs, msg.ID)
		}
	}

	return sub
}

func (s *JSONFile) AckMessage(deviceID string, messageID int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.messageExistsLocked(messageID) {
		return false, nil
	}
	if s.data.MessageAcks == nil {
		s.data.MessageAcks = map[string][]int64{}
	}
	for _, id := range s.data.MessageAcks[deviceID] {
		if id == messageID {
			return true, nil
		}
	}

	s.data.MessageAcks[deviceID] = append(s.data.MessageAcks[deviceID], messageID)
	return true, s.saveLocked()
}

func (s *JSONFile) Telemetry(limit int) []domain.Telemetry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.data.Telemetry
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}

	start := len(items) - limit
	out := append([]domain.Telemetry(nil), items[start:]...)
	reverseTelemetry(out)
	return out
}

func (s *JSONFile) AddTelemetry(t domain.Telemetry) (domain.Telemetry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	t.ID = s.data.NextTelemetryID
	t.ReceivedAt = now
	if t.ReportTimestamp.IsZero() {
		t.ReportTimestamp = now
	}
	if t.App == nil {
		t.App = map[string]any{}
	}

	s.data.NextTelemetryID++
	s.data.Telemetry = append(s.data.Telemetry, t)
	if len(s.data.Telemetry) > maxTelemetry {
		s.data.Telemetry = s.data.Telemetry[len(s.data.Telemetry)-maxTelemetry:]
	}
	return t, s.saveLocked()
}

func (s *JSONFile) load() error {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}

	if err := json.Unmarshal(b, &s.data); err != nil {
		return err
	}
	if s.data.NextMessageID == 0 {
		s.data.NextMessageID = int64(len(s.data.Messages)) + 1
	}
	if s.data.NextTelemetryID == 0 {
		s.data.NextTelemetryID = int64(len(s.data.Telemetry)) + 1
	}
	if s.data.MessageAcks == nil {
		s.data.MessageAcks = map[string][]int64{}
	}
	for i := range s.data.Messages {
		if s.data.Messages[i].Channel == "" {
			s.data.Messages[i].Channel = "default"
		}
	}
	return nil
}

func (s *JSONFile) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func reverseMessages(items []domain.Message) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

func reverseTelemetry(items []domain.Telemetry) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

func (s *JSONFile) messageExistsLocked(id int64) bool {
	for _, msg := range s.data.Messages {
		if msg.ID == id {
			return true
		}
	}
	return false
}
