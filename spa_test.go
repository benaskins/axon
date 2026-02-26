package axon_test

import (
	"embed"
	"net/http/httptest"
	"testing"

	"github.com/benaskins/axon"
)

//go:embed testdata/static/*
var testStaticFS embed.FS

func TestSPAHandler_ServesIndexHTML(t *testing.T) {
	handler := axon.SPAHandler(testStaticFS, "testdata/static")
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSPAHandler_FallbackToIndex(t *testing.T) {
	handler := axon.SPAHandler(testStaticFS, "testdata/static")
	req := httptest.NewRequest("GET", "/some/route", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Error("expected no-cache for SPA fallback")
	}
}

func TestSPAHandler_AppPath404(t *testing.T) {
	handler := axon.SPAHandler(testStaticFS, "testdata/static")
	req := httptest.NewRequest("GET", "/_app/missing.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("expected 404 for missing /_app/ file, got %d", w.Code)
	}
}
