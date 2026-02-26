package axon_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benaskins/axon"
)

func TestRequireAuth_NoCookie(t *testing.T) {
	mockAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("auth service should not be called")
	}))
	defer mockAuth.Close()

	client := axon.NewAuthClientPlain(mockAuth.URL)
	defer client.StopSweep()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})

	handler := axon.RequireAuth(client)(inner)
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequireAuth_ValidSession(t *testing.T) {
	mockAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"user_id":  "user_123",
			"username": "ben",
		})
	}))
	defer mockAuth.Close()

	client := axon.NewAuthClientPlain(mockAuth.URL)
	defer client.StopSweep()

	var gotUserID, gotUsername string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = axon.UserID(r.Context())
		gotUsername = axon.Username(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := axon.RequireAuth(client)(inner)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "valid-token"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if gotUserID != "user_123" {
		t.Errorf("expected user_123, got %s", gotUserID)
	}
	if gotUsername != "ben" {
		t.Errorf("expected ben, got %s", gotUsername)
	}
}

func TestRequireAuth_InvalidSession(t *testing.T) {
	mockAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer mockAuth.Close()

	client := axon.NewAuthClientPlain(mockAuth.URL)
	defer client.StopSweep()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})

	handler := axon.RequireAuth(client)(inner)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "bad-token"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequireAuth_ServiceUnavailable(t *testing.T) {
	client := axon.NewAuthClientPlain("http://localhost:99999")
	defer client.StopSweep()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})

	handler := axon.RequireAuth(client)(inner)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "some-token"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestRequireAuth_CustomCookieName(t *testing.T) {
	mockAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"user_id":  "user_123",
			"username": "ben",
		})
	}))
	defer mockAuth.Close()

	client := axon.NewAuthClientPlain(mockAuth.URL)
	defer client.StopSweep()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := axon.RequireAuth(client, axon.WithCookieName("auth_token"))(inner)

	// Without the right cookie name, should fail
	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "valid-token"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong cookie name, got %d", w.Code)
	}

	// With the right cookie name, should succeed
	req = httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: "valid-token"})
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with correct cookie name, got %d", w.Code)
	}
}

func TestUserID_EmptyContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if got := axon.UserID(req.Context()); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestUsername_EmptyContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if got := axon.Username(req.Context()); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
