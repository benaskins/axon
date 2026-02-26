package axon_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/benaskins/axon"
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
		json.NewEncoder(w).Encode(map[string]any{
			"user_id":  "user_123",
			"username": "ben",
		})
	}))
	defer mockAuth.Close()

	client := axon.NewAuthClientPlain(mockAuth.URL)
	defer client.Close()
	session, err := client.ValidateSession("valid-token")
	if err != nil {
		t.Fatalf("ValidateSession failed: %v", err)
	}
	if session.UserID() != "user_123" {
		t.Errorf("expected user_123, got %s", session.UserID())
	}
	if session.Username() != "ben" {
		t.Errorf("expected username ben, got %s", session.Username())
	}
}

func TestAuthClient_ValidateSession_InvalidToken(t *testing.T) {
	mockAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	}))
	defer mockAuth.Close()

	client := axon.NewAuthClientPlain(mockAuth.URL)
	defer client.Close()
	_, err := client.ValidateSession("invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestAuthClient_ValidateSession_ServiceDown(t *testing.T) {
	client := axon.NewAuthClientPlain("http://localhost:99999")
	defer client.Close()
	_, err := client.ValidateSession("some-token")
	if err == nil {
		t.Fatal("expected error for service down")
	}
}

func TestAuthClient_ValidateSession_CachesResult(t *testing.T) {
	var callCount atomic.Int32
	mockAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"user_id": "user_456", "username": "testuser"})
	}))
	defer mockAuth.Close()

	client := axon.NewAuthClientPlain(mockAuth.URL)
	defer client.Close()

	session, err := client.ValidateSession("cached-token")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if session.UserID() != "user_456" {
		t.Errorf("expected user_456, got %s", session.UserID())
	}

	session, err = client.ValidateSession("cached-token")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if session.UserID() != "user_456" {
		t.Errorf("expected user_456, got %s", session.UserID())
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

	client := axon.NewAuthClientPlain(mockAuth.URL)
	defer client.Close()

	client.ValidateSession("bad-token")
	client.ValidateSession("bad-token")

	if callCount.Load() != 2 {
		t.Errorf("expected 2 server calls (no caching on failure), got %d", callCount.Load())
	}
}

func TestAuthClient_CacheExpiry(t *testing.T) {
	var callCount atomic.Int32
	mockAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"user_id": "user_789"})
	}))
	defer mockAuth.Close()

	client := axon.NewAuthClientPlain(mockAuth.URL, axon.WithCacheTTL(50*time.Millisecond))
	defer client.Close()

	// First call hits server
	_, err := client.ValidateSession("expiring-token")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if callCount.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", callCount.Load())
	}

	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)

	// Second call should hit server again after expiry
	_, err = client.ValidateSession("expiring-token")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if callCount.Load() != 2 {
		t.Errorf("expected 2 calls after cache expiry, got %d", callCount.Load())
	}
}

func TestNewAuthClient_ReturnsErrorOnTLSFailure(t *testing.T) {
	// Without CLIENT_CERT/CLIENT_KEY/CA_CERT set, NewAuthClient should return error
	_, err := axon.NewAuthClient("https://auth.example.com")
	if err == nil {
		t.Fatal("expected error when TLS env vars are not set")
	}
}

func TestAuthClient_Close_Idempotent(t *testing.T) {
	client := axon.NewAuthClientPlain("http://localhost:0")
	client.Close()
	client.Close() // should not panic
}

func TestAuthClient_CustomEndpointPath(t *testing.T) {
	mockAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/check" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"user_id": "custom_user"})
	}))
	defer mockAuth.Close()

	client := axon.NewAuthClientPlain(mockAuth.URL, axon.WithEndpointPath("/auth/check"))
	defer client.Close()

	session, err := client.ValidateSession("token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.UserID() != "custom_user" {
		t.Errorf("expected custom_user, got %s", session.UserID())
	}
}

func TestSessionInfo_Claim(t *testing.T) {
	mockAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"user_id": "u1",
			"role":    "admin",
		})
	}))
	defer mockAuth.Close()

	client := axon.NewAuthClientPlain(mockAuth.URL)
	defer client.Close()

	session, err := client.ValidateSession("token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Claim("role") != "admin" {
		t.Errorf("expected role=admin, got %v", session.Claim("role"))
	}
	if session.Claim("nonexistent") != nil {
		t.Errorf("expected nil for missing claim, got %v", session.Claim("nonexistent"))
	}
}
