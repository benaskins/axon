package axon_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benaskins/axon"
)

func TestRequestLogging_CapturesStatus(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := axon.RequestLogging(inner)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestResponseWriter_Flush(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &axon.ResponseWriter{ResponseWriter: w, StatusCode: 200}
	rw.Flush() // should not panic
}
