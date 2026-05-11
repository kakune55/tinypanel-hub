package httpapi

import (
	"net/http"
	"strings"
)

func (s *Server) handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	channel := strings.TrimSpace(r.PathValue("channel"))
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	limit := queryInt(r, "limit", 20, 1, 100)

	if channel == "" {
		writeError(w, http.StatusBadRequest, "channel is required")
		return
	}
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	sub := s.store.MessageSubscription(deviceID, channel, limit)
	writeJSON(w, http.StatusOK, sub)
}
