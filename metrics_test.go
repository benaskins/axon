package axon_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/benaskins/axon"
)

func TestRequestMetrics_IncreasesCounter(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	mux := http.NewServeMux()
	mux.Handle("GET /test", inner)

	handler := axon.RequestMetrics(mux)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMetricsHandler_ServesPrometheus(t *testing.T) {
	handler := axon.MetricsHandler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequestMetrics_ExposesOTelMetrics(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	mux := http.NewServeMux()
	mux.Handle("GET /api/test", inner)
	handler := axon.RequestMetrics(mux)

	// Make a request to generate metrics
	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Check that metrics appear in Prometheus output
	metricsHandler := axon.MetricsHandler()
	metricsReq := httptest.NewRequest("GET", "/metrics", nil)
	metricsW := httptest.NewRecorder()
	metricsHandler.ServeHTTP(metricsW, metricsReq)

	body := metricsW.Body.String()
	if !strings.Contains(body, "http_server_request_duration") {
		t.Errorf("expected http_server_request_duration in metrics output")
	}
}
