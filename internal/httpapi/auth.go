package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"

	"tinypanel-hub/internal/domain"
)

type contextKey string

const (
	userContextKey   contextKey = "user"
	deviceContextKey contextKey = "device"
)

func (s *Server) adminAuthorized(r *http.Request) bool {
	if s.apiToken == "" {
		return true
	}

	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		token = r.Header.Get("X-API-Token")
	}

	if len(token) != len(s.apiToken) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.apiToken)) == 1
}

func currentUser(r *http.Request) (domain.User, bool) {
	user, ok := r.Context().Value(userContextKey).(domain.User)
	return user, ok
}

func currentDevice(r *http.Request) (domain.Device, bool) {
	device, ok := r.Context().Value(deviceContextKey).(domain.Device)
	return device, ok
}

func withUser(r *http.Request, user domain.User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userContextKey, user))
}

func withDevice(r *http.Request, device domain.Device) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), deviceContextKey, device))
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomSecret() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
