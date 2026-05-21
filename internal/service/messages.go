package service

import "tinypanel-hub/internal/domain"

type MessageService struct {
	store MessageStore
}

func (s MessageService) List(limit int) []domain.Message {
	return s.store.Messages(limit)
}

func (s MessageService) Get(id int64) (domain.Message, bool) {
	return s.store.Message(id)
}

func (s MessageService) Create(channel, author, body string) (domain.Message, error) {
	return s.store.AddMessage(channel, author, body)
}

func (s MessageService) Subscription(deviceID, channel string, limit int) domain.MessageSubscription {
	return s.store.MessageSubscription(deviceID, channel, limit)
}

func (s MessageService) Ack(deviceID string, messageID int64) (bool, error) {
	return s.store.AckMessage(deviceID, messageID)
}
