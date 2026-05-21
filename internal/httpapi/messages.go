package httpapi

import (
	"net/http"
	"strings"

	"tinypanel-hub/internal/domain"
)

func (s *Server) handleListDeviceMessages(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	deviceID := pathString(r, "device_id")
	if _, ok := s.services.Devices.Get(user.ID, deviceID); !ok {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	limit := queryInt(r, "limit", 50, 1, 100)
	writeJSON(w, http.StatusOK, s.services.Messages.DeviceMessages(user.ID, deviceID, limit))
}

func (s *Server) handleCreateDeviceMessage(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	deviceID := pathString(r, "device_id")
	if _, ok := s.services.Devices.Get(user.ID, deviceID); !ok {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}

	var req struct {
		Body     string `json:"body"`
		Priority string `json:"priority"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	req.Priority = strings.TrimSpace(req.Priority)
	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}
	if req.Priority == "" {
		req.Priority = domain.MessagePriorityNormal
	}
	if req.Priority != domain.MessagePriorityNormal && req.Priority != domain.MessagePriorityHigh {
		writeError(w, http.StatusBadRequest, "priority must be normal or high")
		return
	}

	msg, err := s.services.Messages.Create(user.ID, deviceID, user.ID, req.Body, req.Priority)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

func (s *Server) handleDeviceMessages(w http.ResponseWriter, r *http.Request) {
	device, _ := currentDevice(r)
	limit := queryInt(r, "limit", 10, 1, 100)
	writeJSON(w, http.StatusOK, map[string]any{
		"device_id": device.ID,
		"messages":  s.services.Messages.Pending(device.ID, limit),
	})
}

func (s *Server) handleDeviceAckMessages(w http.ResponseWriter, r *http.Request) {
	device, _ := currentDevice(r)
	var req struct {
		MessageIDs []int64 `json:"message_ids"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
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

	result, err := s.services.Messages.AckBatch(device.ID, req.MessageIDs)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
