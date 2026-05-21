package httpapi

import "net/http"

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.services.Snapshot.Get(r.Context())
	if err != nil {
		s.logger.Error("weather provider error", "err", err)
		writeError(w, http.StatusBadGateway, "weather provider error")
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}
