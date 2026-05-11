package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func (s *Server) authorized(r *http.Request) bool {
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

func bearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}
