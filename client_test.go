package axon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternalClient_Get(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/things/123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload{Name: "test", Age: 42})
	}))
	defer srv.Close()

	client := NewInternalClient(srv.URL)
	var got payload
	err := client.Get(context.Background(), "/api/things/123", &got)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "test" || got.Age != 42 {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestInternalClient_Get_NonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, http.StatusNotFound, "not found")
	}))
	defer srv.Close()

	client := NewInternalClient(srv.URL)
	var got map[string]string
	err := client.Get(context.Background(), "/api/things/999", &got)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !IsStatusError(err, http.StatusNotFound) {
		t.Errorf("expected StatusError 404, got: %v", err)
	}
}

func TestInternalClient_Post(t *testing.T) {
	type reqBody struct {
		Task string `json:"task"`
	}
	type respBody struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		var req reqBody
		json.NewDecoder(r.Body).Decode(&req)
		if req.Task != "build" {
			t.Errorf("unexpected task: %s", req.Task)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(respBody{ID: "abc", Status: "queued"})
	}))
	defer srv.Close()

	client := NewInternalClient(srv.URL)
	var got respBody
	err := client.Post(context.Background(), "/api/tasks", reqBody{Task: "build"}, &got)
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if got.ID != "abc" || got.Status != "queued" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestInternalClient_Post_NilResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewInternalClient(srv.URL)
	err := client.Post(context.Background(), "/api/fire", map[string]string{"x": "y"}, nil)
	if err != nil {
		t.Fatalf("Post with nil result: %v", err)
	}
}

func TestInternalClient_Post_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, http.StatusInternalServerError, "boom")
	}))
	defer srv.Close()

	client := NewInternalClient(srv.URL)
	var got map[string]string
	err := client.Post(context.Background(), "/api/tasks", map[string]string{}, &got)
	if err == nil {
		t.Fatal("expected error for 500")
	}
}
