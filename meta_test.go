package axon

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetaMiddleware_ExtractsHeaders(t *testing.T) {
	var gotRunID, gotTraceID string

	handler := MetaHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRunID = Meta(r.Context(), "run-id")
		gotTraceID = Meta(r.Context(), "trace-id")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Axon-Run-Id", "run-20260304")
	req.Header.Set("X-Axon-Trace-Id", "abc123")

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotRunID != "run-20260304" {
		t.Errorf("expected run-id 'run-20260304', got %q", gotRunID)
	}
	if gotTraceID != "abc123" {
		t.Errorf("expected trace-id 'abc123', got %q", gotTraceID)
	}
}

func TestMetaMiddleware_NoHeaders(t *testing.T) {
	var gotRunID string

	handler := MetaHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRunID = Meta(r.Context(), "run-id")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotRunID != "" {
		t.Errorf("expected empty run-id, got %q", gotRunID)
	}
}

func TestMetaMiddleware_IgnoresNonAxonHeaders(t *testing.T) {
	var gotMeta string

	handler := MetaHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMeta = Meta(r.Context(), "authorization")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret")

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotMeta != "" {
		t.Errorf("expected empty meta for non-axon header, got %q", gotMeta)
	}
}

func TestRunID_Shortcut(t *testing.T) {
	var gotRunID string

	handler := MetaHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRunID = RunID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Axon-Run-Id", "run-123")

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotRunID != "run-123" {
		t.Errorf("expected 'run-123', got %q", gotRunID)
	}
}
