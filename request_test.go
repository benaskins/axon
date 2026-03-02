package axon

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON_Success(t *testing.T) {
	type req struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	body := strings.NewReader(`{"name":"alice","age":30}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	got, ok := DecodeJSON[req](w, r)
	if !ok {
		t.Fatalf("expected ok=true, response: %s", w.Body.String())
	}
	if got.Name != "alice" || got.Age != 30 {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	body := strings.NewReader(`{invalid}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	_, ok := DecodeJSON[map[string]string](w, r)
	if ok {
		t.Fatal("expected ok=false for invalid JSON")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDecodeJSON_EmptyBody(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	_, ok := DecodeJSON[map[string]string](w, r)
	if ok {
		t.Fatal("expected ok=false for empty body")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
