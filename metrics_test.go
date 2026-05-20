package axon_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benaskins/axon"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestRequestMetrics_PassesThrough(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	mux := http.NewServeMux()
	mux.Handle("GET /test", inner)

	handler := axon.RequestMetrics(noop.NewMeterProvider())(mux)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
