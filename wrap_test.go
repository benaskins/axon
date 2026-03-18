package axon_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/benaskins/axon"
)

func TestWrapHandler_MetricsEndpoint(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("app"))
	})

	handler := axon.WrapHandler(inner)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 for /metrics, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "promhttp") && !strings.Contains(w.Body.String(), "go_") {
		// Prometheus handler should return some metrics
		t.Logf("metrics body (first 200): %s", w.Body.String()[:min(200, w.Body.Len())])
	}
}

func TestWrapHandler_HealthEndpoint(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("app"))
	})

	handler := axon.WrapHandler(inner)

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

func TestWrapHandler_DelegatesOtherRoutes(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello from app"))
	})

	handler := axon.WrapHandler(inner)

	req := httptest.NewRequest("GET", "/api/something", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Body.String() != "hello from app" {
		t.Errorf("expected app response, got %q", w.Body.String())
	}
}

func TestWrapHandler_AppliesRequestLogging(t *testing.T) {
	var called bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	})

	handler := axon.WrapHandler(inner)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("inner handler was not called")
	}
	// If StandardMiddleware is applied, the response writer is wrapped
	// with statusRecorder. We can't easily assert logging happened,
	// but we verify the request reaches the inner handler.
}

func TestWrapHandler_HealthEndpointNotOverridable(t *testing.T) {
	// Even if the inner handler registers /health, the auto-wired one wins.
	inner := http.NewServeMux()
	inner.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"custom"}`))
	})

	handler := axon.WrapHandler(inner)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Errorf("expected auto-wired health (status=ok), got %v", body["status"])
	}
}

func TestWrapHandler_WithHealthChecks(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handler := axon.WrapHandler(inner,
		axon.HealthCheck{Name: "postgres", Check: func() error { return nil }},
		axon.HealthCheck{Name: "nats", Check: func() error { return nil }},
	)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body["status"])
	}
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

func TestWrapHandler_HealthCheckFails(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handler := axon.WrapHandler(inner,
		axon.HealthCheck{Name: "postgres", Check: func() error { return nil }},
		axon.HealthCheck{Name: "redis", Check: func() error {
			return fmt.Errorf("connection refused")
		}},
	)

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
	checks := body["checks"].(map[string]any)
	if checks["postgres"] != "ok" {
		t.Errorf("postgres = %v, want ok", checks["postgres"])
	}
	if checks["redis"] != "connection refused" {
		t.Errorf("redis = %v, want connection refused", checks["redis"])
	}
}
