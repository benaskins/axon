package axon

import (
	"errors"
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

func TestDecodeJSON_OversizedBody(t *testing.T) {
	// Default limit is 1MB; send 2MB
	big := strings.Repeat("x", 2<<20)
	body := strings.NewReader(`{"name":"` + big + `"}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	type req struct {
		Name string `json:"name"`
	}
	_, ok := DecodeJSON[req](w, r)
	if ok {
		t.Fatal("expected ok=false for oversized body")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

type validReq struct {
	Name string `json:"name"`
}

func (v validReq) Validate() error {
	if v.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

func TestDecodeJSON_Validatable_Pass(t *testing.T) {
	body := strings.NewReader(`{"name":"alice"}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	got, ok := DecodeJSON[validReq](w, r)
	if !ok {
		t.Fatalf("expected ok=true, response: %s", w.Body.String())
	}
	if got.Name != "alice" {
		t.Errorf("unexpected name: %s", got.Name)
	}
}

func TestDecodeJSON_Validatable_Fail(t *testing.T) {
	body := strings.NewReader(`{"name":""}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	_, ok := DecodeJSON[validReq](w, r)
	if ok {
		t.Fatal("expected ok=false for failed validation")
	}
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "name is required") {
		t.Errorf("expected validation message in body, got: %s", w.Body.String())
	}
}
