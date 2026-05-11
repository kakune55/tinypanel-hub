package httpapi

import (
	"net/http"
)

func (s *Server) handleGetWeather(w http.ResponseWriter, r *http.Request) {
	if s.weather != nil {
		weather, err := s.weather.Current(r.Context())
		if err != nil {
			s.logger.Error("weather provider error", "err", err)
			writeError(w, http.StatusBadGateway, "weather provider error")
			return
		}
		writeJSON(w, http.StatusOK, weather)
		return
	}
	writeJSON(w, http.StatusOK, s.store.Weather())
}
