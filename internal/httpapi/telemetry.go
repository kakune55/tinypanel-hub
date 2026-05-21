package httpapi

import (
	"net/http"
	"strings"

	"tinypanel-hub/internal/domain"
)

func (s *Server) handleGetTelemetry(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50, 1, 500)
	writeJSON(w, http.StatusOK, s.services.Telemetry.List(limit))
}

func (s *Server) handlePostTelemetry(w http.ResponseWriter, r *http.Request) {
	var req domain.Telemetry
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	req.DeviceID = strings.TrimSpace(req.DeviceID)
	req.BootID = strings.TrimSpace(req.BootID)
	req.Power.Battery.Status = strings.TrimSpace(req.Power.Battery.Status)
	req.Network.SSID = strings.TrimSpace(req.Network.SSID)
	req.Network.IP = strings.TrimSpace(req.Network.IP)
	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	if req.SchemaVersion != 1 {
		writeError(w, http.StatusBadRequest, "schema_version must be 1")
		return
	}
	if req.BootID == "" {
		writeError(w, http.StatusBadRequest, "boot_id is required")
		return
	}
	if req.Sequence < 0 {
		writeError(w, http.StatusBadRequest, "sequence must be greater than or equal to 0")
		return
	}
	if req.ReportTimestamp.IsZero() {
		writeError(w, http.StatusBadRequest, "report_timestamp is required")
		return
	}

	item, err := s.services.Telemetry.Create(req)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
