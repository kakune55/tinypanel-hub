package httpapi

import (
	"net/http"
	"strings"
)

const defaultMessageChannel = "default"

func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 20, 1, 100)
	writeJSON(w, http.StatusOK, s.services.Messages.List(limit))
}

func (s *Server) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Channel string `json:"channel"`
		Author  string `json:"author"`
		Body    string `json:"body"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	req.Channel = strings.TrimSpace(req.Channel)
	req.Author = strings.TrimSpace(req.Author)
	req.Body = strings.TrimSpace(req.Body)
	if req.Channel == "" {
		req.Channel = defaultMessageChannel
	}
	if req.Author == "" {
		req.Author = "anonymous"
	}
	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}

	msg, err := s.services.Messages.Create(req.Channel, req.Author, req.Body)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

func (s *Server) handleGetMessage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	msg, found := s.services.Messages.Get(id)
	if !found {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}
	writeJSON(w, http.StatusOK, msg)
}

func (s *Server) handleAckMessage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	var req struct {
		DeviceID string `json:"device_id"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	req.DeviceID = requestDeviceID(r, req.DeviceID)
	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	found, err := s.services.Messages.Ack(req.DeviceID, id)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"device_id":  req.DeviceID,
		"message_id": id,
		"acked":      true,
	})
}

func (s *Server) handleAckMessages(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID   string  `json:"device_id"`
		MessageIDs []int64 `json:"message_ids"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	req.DeviceID = requestDeviceID(r, req.DeviceID)
	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	if len(req.MessageIDs) == 0 {
		writeError(w, http.StatusBadRequest, "message_ids is required")
		return
	}
	if len(req.MessageIDs) > 100 {
		writeError(w, http.StatusBadRequest, "message_ids must not exceed 100 items")
		return
	}
	for _, id := range req.MessageIDs {
		if id <= 0 {
			writeError(w, http.StatusBadRequest, "message_ids must contain positive integers")
			return
		}
	}

	result, err := s.services.Messages.AckBatch(req.DeviceID, req.MessageIDs)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
