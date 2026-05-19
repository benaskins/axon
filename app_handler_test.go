package axon_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/benaskins/axon"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestAppHandler_HealthEndpoint(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("app"))
	})

	handler := axon.AppHandler(inner, axon.HandlerOptions{})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 for /health, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body["status"])
	}
}

func TestAppHandler_DelegatesOtherRoutes(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello from app"))
	})

	handler := axon.AppHandler(inner, axon.HandlerOptions{})

	req := httptest.NewRequest("GET", "/api/something", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Body.String() != "hello from app" {
		t.Errorf("expected app response, got %q", w.Body.String())
	}
}

func TestAppHandler_HealthEndpointNotOverridable(t *testing.T) {
	inner := http.NewServeMux()
	inner.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"custom"}`))
	})

	handler := axon.AppHandler(inner, axon.HandlerOptions{})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Errorf("expected auto-wired health (status=ok), got %v", body["status"])
	}
}

func TestAppHandler_HealthChecks(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handler := axon.AppHandler(inner, axon.HandlerOptions{
		HealthChecks: []axon.HealthCheck{
			{Name: "postgres", Check: func() error { return nil }},
			{Name: "nats", Check: func() error { return nil }},
		},
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	checks, ok := body["checks"].(map[string]any)
	if !ok {
		t.Fatalf("expected checks map, got %T", body["checks"])
	}
	if checks["postgres"] != "ok" {
		t.Errorf("postgres = %v, want ok", checks["postgres"])
	}
	if checks["nats"] != "ok" {
		t.Errorf("nats = %v, want ok", checks["nats"])
	}
}

func TestAppHandler_HealthCheckFails(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handler := axon.AppHandler(inner, axon.HandlerOptions{
		HealthChecks: []axon.HealthCheck{
			{Name: "redis", Check: func() error { return fmt.Errorf("connection refused") }},
		},
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 503 {
		t.Errorf("expected 503, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "unhealthy" {
		t.Errorf("expected status=unhealthy, got %v", body["status"])
	}
}

func TestAppHandler_UsesProvidedLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	handler := axon.AppHandler(inner, axon.HandlerOptions{Logger: logger})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !strings.Contains(buf.String(), `"path":"/api/test"`) {
		t.Errorf("expected log line with path, got %q", buf.String())
	}
}

func TestAppHandler_UsesProvidedMeterProvider(t *testing.T) {
	// noop provider proves the middleware accepts injection without panicking.
	mp := noop.NewMeterProvider()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	handler := axon.AppHandler(inner, axon.HandlerOptions{MeterProvider: mp})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAppHandler_NilOptionsUsesDefaults(t *testing.T) {
	// Zero-value HandlerOptions should work and fall back to slog.Default
	// and otel.GetMeterProvider (noop by default).
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	handler := axon.AppHandler(inner, axon.HandlerOptions{})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
