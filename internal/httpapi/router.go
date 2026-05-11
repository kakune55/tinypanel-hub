package httpapi

import "net/http"

const apiPrefix = "/api/v1/"

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)

	s.mux.HandleFunc("GET /api/v1/snapshot", s.handleSnapshot)
	s.mux.HandleFunc("GET /api/v1/weather", s.handleGetWeather)
	s.mux.HandleFunc("GET /api/v1/messages", s.handleGetMessages)
	s.mux.HandleFunc("POST /api/v1/messages", s.handlePostMessage)
	s.mux.HandleFunc("GET /api/v1/messages/{id}", s.handleGetMessage)
	s.mux.HandleFunc("POST /api/v1/messages/{id}/ack", s.handleAckMessage)
	s.mux.HandleFunc("GET /api/v1/subscriptions/{channel}", s.handleGetSubscription)
	s.mux.HandleFunc("GET /api/v1/telemetry", s.handleGetTelemetry)
	s.mux.HandleFunc("POST /api/v1/telemetry", s.handlePostTelemetry)
}

func isAPIRoute(path string) bool {
	return len(path) >= len(apiPrefix) && path[:len(apiPrefix)] == apiPrefix
}

var _ http.Handler = (*Server)(nil)
