package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"tinypanel-hub/internal/service"
)

type Options struct {
	APIToken        string
	WeatherProvider WeatherProvider
	WebHandler      http.Handler
}

type WeatherProvider = service.WeatherProvider
type Store = service.Store

type Server struct {
	services service.Services
	logger   *slog.Logger
	mux      chi.Router
	apiToken string
	web      http.Handler
}

func New(store Store, logger *slog.Logger, opts Options) http.Handler {
	s := &Server{
		services: service.New(store, opts.WeatherProvider),
		logger:   logger,
		mux:      chi.NewRouter(),
		apiToken: opts.APIToken,
		web:      opts.WebHandler,
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) writeStoreError(w http.ResponseWriter, err error) {
	s.logger.Error("store error", "err", err)
	writeError(w, http.StatusInternalServerError, "store error")
}
