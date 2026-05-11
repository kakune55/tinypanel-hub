package store

import (
	"time"

	"tinypanel-hub/internal/domain"
)

func (s *FileStore) Snapshot() domain.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	telemetry, err := s.telemetry.loadRecent(maxTelemetry)
	if err != nil {
		telemetry = nil
	}
	return domain.Snapshot{
		Weather:   s.state.data.Weather,
		Messages:  append([]domain.Message(nil), s.state.data.Messages...),
		Telemetry: telemetry,
	}
}

func (s *FileStore) Weather() domain.Weather {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.data.Weather
}

func (s *FileStore) Messages(limit int) []domain.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.state.data.Messages
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}

	start := len(items) - limit
	out := append([]domain.Message(nil), items[start:]...)
	reverseMessages(out)
	return out
}

func (s *FileStore) Message(id int64) (domain.Message, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, msg := range s.state.data.Messages {
		if msg.ID == id {
			return msg, true
		}
	}
	return domain.Message{}, false
}

func (s *FileStore) AddMessage(channel, author, body string) (domain.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := domain.Message{
		ID:        s.state.data.NextMessageID,
		Channel:   channel,
		Author:    author,
		Body:      body,
		CreatedAt: time.Now().UTC(),
	}
	s.state.data.NextMessageID++
	s.state.data.Messages = append(s.state.data.Messages, msg)
	if len(s.state.data.Messages) > maxMessages {
		s.state.data.Messages = s.state.data.Messages[len(s.state.data.Messages)-maxMessages:]
	}
	return msg, s.state.save()
}

func (s *FileStore) MessageSubscription(deviceID, channel string, limit int) domain.MessageSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	acked := make(map[int64]bool)
	for _, id := range s.state.data.MessageAcks[deviceID] {
		acked[id] = true
	}

	sub := domain.MessageSubscription{
		DeviceID:   deviceID,
		Channel:    channel,
		MessageIDs: []int64{},
	}

	for _, msg := range s.state.data.Messages {
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

func (s *FileStore) AckMessage(deviceID string, messageID int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.messageExists(messageID) {
		return false, nil
	}
	if s.state.data.MessageAcks == nil {
		s.state.data.MessageAcks = map[string][]int64{}
	}
	for _, id := range s.state.data.MessageAcks[deviceID] {
		if id == messageID {
			return true, nil
		}
	}

	s.state.data.MessageAcks[deviceID] = append(s.state.data.MessageAcks[deviceID], messageID)
	return true, s.state.save()
}

func (s *FileStore) messageExists(id int64) bool {
	for _, msg := range s.state.data.Messages {
		if msg.ID == id {
			return true
		}
	}
	return false
}

func reverseMessages(items []domain.Message) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}
