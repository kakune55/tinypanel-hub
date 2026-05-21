package httpapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (s *Server) routes() {
	s.mux.Use(middleware.RequestID)
	s.mux.Use(middleware.RealIP)
	s.mux.Use(middleware.Recoverer)
	s.mux.Use(noSniff)
	s.mux.Use(s.logRequest)

	s.mux.Get("/healthz", s.handleHealth)

	s.mux.Route("/api/v1", func(r chi.Router) {
		r.Use(s.requireAuth)

		r.Get("/snapshot", s.handleSnapshot)
		r.Get("/weather", s.handleGetWeather)

		r.Route("/messages", func(r chi.Router) {
			r.Get("/", s.handleGetMessages)
			r.Post("/", s.handlePostMessage)
			r.Post("/ack", s.handleAckMessages)
			r.Get("/{id}", s.handleGetMessage)
			r.Post("/{id}/ack", s.handleAckMessage)
		})

		r.Get("/subscriptions/{channel}", s.handleGetSubscription)

		r.Route("/todos", func(r chi.Router) {
			r.Get("/", s.handleGetTodos)
			r.Post("/", s.handlePostTodo)
			r.Get("/{id}", s.handleGetTodo)
			r.Patch("/{id}", s.handlePatchTodo)
			r.Delete("/{id}", s.handleDeleteTodo)
		})

		r.Get("/telemetry", s.handleGetTelemetry)
		r.Post("/telemetry", s.handlePostTelemetry)
		r.Post("/telemetry/batch", s.handlePostTelemetryBatch)
	})

	s.mux.NotFound(s.handleNotFound)
}

var _ http.Handler = (*Server)(nil)

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") || s.web == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	s.web.ServeHTTP(w, r)
}
