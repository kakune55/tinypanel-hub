package service

import "tinypanel-hub/internal/domain"

type MessageAckBatchResult struct {
	DeviceID   string  `json:"device_id"`
	AckedIDs   []int64 `json:"acked_ids"`
	MissingIDs []int64 `json:"missing_ids,omitempty"`
}

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

func (s MessageService) AckBatch(deviceID string, messageIDs []int64) (MessageAckBatchResult, error) {
	result := MessageAckBatchResult{
		DeviceID: deviceID,
		AckedIDs: []int64{},
	}
	for _, id := range messageIDs {
		found, err := s.store.AckMessage(deviceID, id)
		if err != nil {
			return result, err
		}
		if !found {
			result.MissingIDs = append(result.MissingIDs, id)
			continue
		}
		result.AckedIDs = append(result.AckedIDs, id)
	}
	return result, nil
}
