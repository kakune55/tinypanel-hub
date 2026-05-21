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
		r.Route("/admin", func(r chi.Router) {
			r.Use(s.requireAdminAuth)
			r.Post("/users", s.handleCreateUser)
		})

		r.Post("/device/hello", s.handleDeviceHello)
		r.Route("/device", func(r chi.Router) {
			r.Use(s.requireDeviceAuth)
			r.Get("/messages", s.handleDeviceMessages)
			r.Post("/messages/ack", s.handleDeviceAckMessages)
			r.Get("/todos", s.handleDeviceTodos)
			r.Post("/telemetry", s.handleDeviceTelemetry)
			r.Post("/telemetry/batch", s.handleDeviceTelemetryBatch)
			r.Get("/snapshot", s.handleDeviceSnapshot)
		})

		r.Group(func(r chi.Router) {
			r.Use(s.requireUserAuth)
			r.Get("/me", s.handleMe)
			r.Get("/snapshot", s.handleSnapshot)
			r.Get("/weather", s.handleGetWeather)

			r.Route("/devices", func(r chi.Router) {
				r.Get("/", s.handleListDevices)
				r.Post("/bind", s.handleBindDevice)
				r.Get("/{device_id}", s.handleGetDevice)
				r.Patch("/{device_id}", s.handlePatchDevice)
				r.Delete("/{device_id}", s.handleDeleteDevice)
				r.Get("/{device_id}/messages", s.handleListDeviceMessages)
				r.Post("/{device_id}/messages", s.handleCreateDeviceMessage)
				r.Get("/{device_id}/telemetry", s.handleGetDeviceTelemetry)
			})

			r.Route("/todos", func(r chi.Router) {
				r.Get("/", s.handleGetTodos)
				r.Post("/", s.handlePostTodo)
				r.Get("/{id}", s.handleGetTodo)
				r.Patch("/{id}", s.handlePatchTodo)
				r.Delete("/{id}", s.handleDeleteTodo)
			})
		})
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
