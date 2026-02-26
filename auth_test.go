package axon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestAuthClient_ValidateSession_Success(t *testing.T) {
	mockAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/validate" {
			t.Errorf("expected /api/validate, got %s", r.URL.Path)
		}

		cookie, err := r.Cookie("session")
		if err != nil || cookie.Value != "valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"user_id":  "user_123",
			"username": "ben",
		})
	}))
	defer mockAuth.Close()

	client := NewAuthClientPlain(mockAuth.URL)
	defer client.StopSweep()
	session, err := client.ValidateSession("valid-token")
	if err != nil {
		t.Fatalf("ValidateSession failed: %v", err)
	}
	if session.UserID != "user_123" {
		t.Errorf("expected user_123, got %s", session.UserID)
	}
	if session.Username != "ben" {
		t.Errorf("expected username ben, got %s", session.Username)
	}
}

func TestAuthClient_ValidateSession_InvalidToken(t *testing.T) {
	mockAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	}))
	defer mockAuth.Close()

	client := NewAuthClientPlain(mockAuth.URL)
	defer client.StopSweep()
	_, err := client.ValidateSession("invalid-token")
	if err != ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAuthClient_ValidateSession_ServiceDown(t *testing.T) {
	client := NewAuthClientPlain("http://localhost:99999")
	defer client.StopSweep()
	_, err := client.ValidateSession("some-token")
	if err != ErrServiceUnavailable {
		t.Errorf("expected ErrServiceUnavailable, got %v", err)
	}
}

func TestAuthClient_ValidateSession_CachesResult(t *testing.T) {
	var callCount atomic.Int32
	mockAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"user_id": "user_456", "username": "testuser"})
	}))
	defer mockAuth.Close()

	client := NewAuthClientPlain(mockAuth.URL)
	defer client.StopSweep()

	session, err := client.ValidateSession("cached-token")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if session.UserID != "user_456" {
		t.Errorf("expected user_456, got %s", session.UserID)
	}

	session, err = client.ValidateSession("cached-token")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if session.UserID != "user_456" {
		t.Errorf("expected user_456, got %s", session.UserID)
	}

	if callCount.Load() != 1 {
		t.Errorf("expected 1 server call (cached), got %d", callCount.Load())
	}
}

func TestAuthClient_ValidateSession_DoesNotCacheFailure(t *testing.T) {
	var callCount atomic.Int32
	mockAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer mockAuth.Close()

	client := NewAuthClientPlain(mockAuth.URL)
	defer client.StopSweep()

	client.ValidateSession("bad-token")
	client.ValidateSession("bad-token")

	if callCount.Load() != 2 {
		t.Errorf("expected 2 server calls (no caching on failure), got %d", callCount.Load())
	}
}

func TestAuthClient_SweepEvictsExpiredEntries(t *testing.T) {
	client := &AuthClient{
		stopSweep: make(chan struct{}),
	}

	client.cache.Store("expired-token", cachedSession{
		userID:    "user_old",
		username:  "old",
		expiresAt: time.Now().Add(-1 * time.Minute),
	})
	client.cache.Store("valid-token", cachedSession{
		userID:    "user_new",
		username:  "new",
		expiresAt: time.Now().Add(5 * time.Minute),
	})

	now := time.Now()
	client.cache.Range(func(key, value any) bool {
		if now.After(value.(cachedSession).expiresAt) {
			client.cache.Delete(key)
		}
		return true
	})

	if _, ok := client.cache.Load("expired-token"); ok {
		t.Error("expected expired-token to be evicted")
	}

	if _, ok := client.cache.Load("valid-token"); !ok {
		t.Error("expected valid-token to remain in cache")
	}
}
