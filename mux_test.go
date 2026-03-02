package axon_test

import (
	"net/http/httptest"
	"testing"

	"github.com/benaskins/axon"
)

func TestNewServiceMux_HasHealthEndpoint(t *testing.T) {
	mux := axon.NewServiceMux(nil)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 for /health, got %d", w.Code)
	}
}

func TestNewServiceMux_HasMetricsEndpoint(t *testing.T) {
	mux := axon.NewServiceMux(nil)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 for /metrics, got %d", w.Code)
	}
}
