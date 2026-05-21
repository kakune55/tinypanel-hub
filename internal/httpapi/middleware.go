package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func noSniff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAdminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.adminAuthorized(r) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="tinypanel-hub"`)
			writeError(w, http.StatusUnauthorized, "missing or invalid admin token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireUserAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing or invalid user token")
			return
		}
		user, ok := s.services.Users.ByTokenHash(tokenHash(token))
		if !ok {
			writeError(w, http.StatusUnauthorized, "missing or invalid user token")
			return
		}
		next.ServeHTTP(w, withUser(r, user))
	})
}

func (s *Server) requireDeviceAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deviceID := r.Header.Get("X-Device-ID")
		secret := r.Header.Get("X-Device-Secret")
		if deviceID == "" || secret == "" {
			writeError(w, http.StatusUnauthorized, "missing or invalid device credentials")
			return
		}
		device, ok := s.services.Devices.ByCredentials(deviceID, tokenHash(secret))
		if !ok {
			writeError(w, http.StatusUnauthorized, "missing or invalid device credentials")
			return
		}
		next.ServeHTTP(w, withDevice(r, device))
	})
}

func (s *Server) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		s.logger.Info("http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status),
			slog.Duration("duration", time.Since(start)),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
