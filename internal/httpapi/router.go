package httpapi

import (
	"net/http"

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
	})
}

var _ http.Handler = (*Server)(nil)
