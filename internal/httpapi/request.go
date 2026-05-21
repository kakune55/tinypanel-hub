package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func pathString(r *http.Request, name string) string {
	return strings.TrimSpace(chi.URLParam(r, name))
}

func requestDeviceID(r *http.Request, fallback string) string {
	if id := strings.TrimSpace(fallback); id != "" {
		return id
	}
	return strings.TrimSpace(r.Header.Get("X-Device-ID"))
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(pathString(r, name), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, name+" must be a positive integer")
		return 0, false
	}
	return id, true
}

func queryInt(r *http.Request, name string, fallback, min, max int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
