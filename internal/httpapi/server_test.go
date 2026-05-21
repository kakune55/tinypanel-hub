package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"tinypanel-hub/internal/store"
)

func TestUserDeviceMessageFlow(t *testing.T) {
	handler := newTestHandler(t)

	userToken := createUser(t, handler, "Alice")
	hello := deviceHello(t, handler, "tinypanel-001", "")
	deviceSecret := hello["device_secret"].(string)
	bindCode := hello["bind_code"].(string)

	rec := userJSON(t, handler, http.MethodPost, "/api/v1/devices/bind", userToken, `{"bind_code":"`+bindCode+`","name":"书桌屏幕"}`, http.StatusOK)
	if !strings.Contains(rec.Body.String(), `"owner_id"`) {
		t.Fatalf("unexpected bind response: %s", rec.Body.String())
	}

	userJSON(t, handler, http.MethodPost, "/api/v1/devices/tinypanel-001/messages", userToken, `{"body":"hello","priority":"normal"}`, http.StatusCreated)

	rec = deviceJSON(t, handler, http.MethodGet, "/api/v1/device/messages", "tinypanel-001", deviceSecret, "", http.StatusOK)
	if !strings.Contains(rec.Body.String(), `"body":"hello"`) {
		t.Fatalf("unexpected pending messages: %s", rec.Body.String())
	}

	deviceJSON(t, handler, http.MethodPost, "/api/v1/device/messages/ack", "tinypanel-001", deviceSecret, `{"message_ids":[1]}`, http.StatusOK)
	rec = deviceJSON(t, handler, http.MethodGet, "/api/v1/device/messages", "tinypanel-001", deviceSecret, "", http.StatusOK)
	if strings.Contains(rec.Body.String(), `"body":"hello"`) {
		t.Fatalf("acked message still pending: %s", rec.Body.String())
	}
}

func TestUserDeviceIsolation(t *testing.T) {
	handler := newTestHandler(t)

	aliceToken := createUser(t, handler, "Alice")
	bobToken := createUser(t, handler, "Bob")
	hello := deviceHello(t, handler, "tinypanel-001", "")
	bindCode := hello["bind_code"].(string)
	userJSON(t, handler, http.MethodPost, "/api/v1/devices/bind", aliceToken, `{"bind_code":"`+bindCode+`","name":"书桌屏幕"}`, http.StatusOK)

	userJSON(t, handler, http.MethodGet, "/api/v1/devices/tinypanel-001", bobToken, "", http.StatusNotFound)
	userJSON(t, handler, http.MethodPost, "/api/v1/devices/tinypanel-001/messages", bobToken, `{"body":"bad"}`, http.StatusNotFound)
}

func TestAuthFailures(t *testing.T) {
	handler := newTestHandler(t)

	userJSON(t, handler, http.MethodGet, "/api/v1/me", "bad-token", "", http.StatusUnauthorized)
	deviceJSON(t, handler, http.MethodGet, "/api/v1/device/messages", "tinypanel-001", "bad-secret", "", http.StatusUnauthorized)
}

func TestDeviceTelemetryBatch(t *testing.T) {
	handler := newTestHandler(t)
	userToken := createUser(t, handler, "Alice")
	hello := deviceHello(t, handler, "tinypanel-001", "")
	deviceSecret := hello["device_secret"].(string)
	bindCode := hello["bind_code"].(string)
	userJSON(t, handler, http.MethodPost, "/api/v1/devices/bind", userToken, `{"bind_code":"`+bindCode+`","name":"书桌屏幕"}`, http.StatusOK)

	body := `{"items":[{"schema_version":1,"boot_id":"boot","sequence":1,"report_timestamp":"2026-05-10T16:20:00+08:00","power":{"battery":{"status":"discharging"}},"environment":{"shtc3":{}},"network":{},"system":{},"storage":{},"app":{}}]}`
	rec := deviceJSON(t, handler, http.MethodPost, "/api/v1/device/telemetry/batch", "tinypanel-001", deviceSecret, body, http.StatusCreated)
	if !strings.Contains(rec.Body.String(), `"count":1`) {
		t.Fatalf("unexpected telemetry response: %s", rec.Body.String())
	}
}

func TestWebFallbackDoesNotCatchAPI(t *testing.T) {
	web := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("web ui"))
	})
	s, err := store.OpenFiles(filepath.Join(t.TempDir(), "state.json"), filepath.Join(t.TempDir(), "telemetry.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	handler := New(s, slog.Default(), Options{APIToken: "admin-token", WebHandler: web})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "web ui" {
		t.Fatalf("unexpected web response: status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound || strings.Contains(rec.Body.String(), "web ui") {
		t.Fatalf("unexpected api missing response: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	s, err := store.OpenFiles(filepath.Join(dir, "state.json"), filepath.Join(dir, "telemetry.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	return New(s, slog.Default(), Options{APIToken: "admin-token"})
}

func createUser(t *testing.T, handler http.Handler, name string) string {
	t.Helper()
	token := strings.ToLower(name) + "-token"
	adminJSON(t, handler, http.MethodPost, "/api/v1/admin/users", `{"name":"`+name+`","api_token":"`+token+`"}`, http.StatusCreated)
	return token
}

func deviceHello(t *testing.T, handler http.Handler, deviceID, secret string) map[string]any {
	t.Helper()
	rec := deviceJSON(t, handler, http.MethodPost, "/api/v1/device/hello", deviceID, secret, "", http.StatusOK)
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func adminJSON(t *testing.T, handler http.Handler, method, path, body string, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer admin-token")
	req.Header.Set("Content-Type", "application/json")
	return serveWant(t, handler, req, wantStatus)
}

func userJSON(t *testing.T, handler http.Handler, method, path, token, body string, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return serveWant(t, handler, req, wantStatus)
}

func deviceJSON(t *testing.T, handler http.Handler, method, path, deviceID, secret, body string, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	req.Header.Set("X-Device-ID", deviceID)
	if secret != "" {
		req.Header.Set("X-Device-Secret", secret)
	}
	req.Header.Set("Content-Type", "application/json")
	return serveWant(t, handler, req, wantStatus)
}

func serveWant(t *testing.T, handler http.Handler, req *http.Request, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s status=%d want=%d body=%s", req.Method, req.URL.Path, rec.Code, wantStatus, rec.Body.String())
	}
	return rec
}
