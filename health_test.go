package axon_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/benaskins/axon"
)

func TestHealthHandler_NoDB(t *testing.T) {
	handler := axon.HealthHandler(nil)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "healthy" {
		t.Errorf("expected healthy, got %s", resp["status"])
	}
	if _, ok := resp["database"]; ok {
		t.Error("database field should not be present without db")
	}
}
