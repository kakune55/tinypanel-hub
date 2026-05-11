package httpapi

import "net/http"

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot := s.store.Snapshot()
	if s.weather != nil {
		weather, err := s.weather.Current(r.Context())
		if err != nil {
			s.logger.Error("weather provider error", "err", err)
			writeError(w, http.StatusBadGateway, "weather provider error")
			return
		}
		snapshot.Weather = weather
	}
	writeJSON(w, http.StatusOK, snapshot)
}
