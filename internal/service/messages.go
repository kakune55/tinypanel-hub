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
	return nil
}

func (s MessageService) Get(id int64) (domain.Message, bool) {
	return domain.Message{}, false
}

func (s MessageService) DeviceMessages(ownerID, deviceID string, limit int) []domain.Message {
	return s.store.DeviceMessages(ownerID, deviceID, limit)
}

func (s MessageService) Create(ownerID, deviceID, authorID, body, priority string) (domain.Message, error) {
	return s.store.AddDeviceMessage(ownerID, deviceID, authorID, body, priority)
}

func (s MessageService) Ack(deviceID string, messageID int64) (bool, error) {
	acked, missing, err := s.store.AckDeviceMessages(deviceID, []int64{messageID})
	return len(acked) == 1 && len(missing) == 0, err
}

func (s MessageService) Pending(deviceID string, limit int) []domain.Message {
	return s.store.PendingDeviceMessages(deviceID, limit)
}

func (s MessageService) AckBatch(deviceID string, messageIDs []int64) (MessageAckBatchResult, error) {
	acked, missing, err := s.store.AckDeviceMessages(deviceID, messageIDs)
	return MessageAckBatchResult{DeviceID: deviceID, AckedIDs: acked, MissingIDs: missing}, err
}
