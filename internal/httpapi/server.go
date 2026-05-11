package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"tinypanel-hub/internal/domain"
)

type Options struct {
	APIToken        string
	WeatherProvider WeatherProvider
}

type WeatherProvider interface {
	Current(context.Context) (domain.Weather, error)
}

type Store interface {
	Snapshot() domain.Snapshot
	Weather() domain.Weather
	Messages(limit int) []domain.Message
	Message(id int64) (domain.Message, bool)
	AddMessage(channel, author, body string) (domain.Message, error)
	MessageSubscription(deviceID, channel string, limit int) domain.MessageSubscription
	AckMessage(deviceID string, messageID int64) (bool, error)
	Telemetry(limit int) []domain.Telemetry
	AddTelemetry(domain.Telemetry) (domain.Telemetry, error)
}

type Server struct {
	store    Store
	logger   *slog.Logger
	mux      *http.ServeMux
	apiToken string
	weather  WeatherProvider
}

func New(store Store, logger *slog.Logger, opts Options) http.Handler {
	s := &Server{
		store:    store,
		logger:   logger,
		mux:      http.NewServeMux(),
		apiToken: opts.APIToken,
		weather:  opts.WeatherProvider,
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if isAPIRoute(r.URL.Path) && !s.authorized(r) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="tinypanel-hub"`)
		writeError(w, http.StatusUnauthorized, "missing or invalid api token")
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) writeStoreError(w http.ResponseWriter, err error) {
	s.logger.Error("store error", "err", err)
	writeError(w, http.StatusInternalServerError, "store error")
}
