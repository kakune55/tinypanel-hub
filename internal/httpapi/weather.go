package httpapi

import (
	"net/http"
)

func (s *Server) handleGetWeather(w http.ResponseWriter, r *http.Request) {
	weather, err := s.services.Weather.Get(r.Context())
	if err != nil {
		s.logger.Error("weather provider error", "err", err)
		writeError(w, http.StatusBadGateway, "weather provider error")
		return
	}
	writeJSON(w, http.StatusOK, weather)
}
