package sse

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetSSEHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	SetSSEHeaders(w)

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %q", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %q", cc)
	}
}

func TestSendEvent(t *testing.T) {
	w := httptest.NewRecorder()

	type testPayload struct {
		Type string `json:"type"`
		ID   int    `json:"id"`
	}

	err := SendEvent(w, w, testPayload{Type: "test", ID: 42})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := w.Body.String()
	if !strings.HasPrefix(body, "data: ") {
		t.Fatalf("expected SSE data frame, got %q", body)
	}
	if !strings.HasSuffix(body, "\n\n") {
		t.Fatalf("expected double newline suffix, got %q", body)
	}

	// Parse the JSON payload
	jsonStr := strings.TrimPrefix(body, "data: ")
	jsonStr = strings.TrimSpace(jsonStr)
	var ev testPayload
	if err := json.Unmarshal([]byte(jsonStr), &ev); err != nil {
		t.Fatalf("failed to parse event JSON: %v", err)
	}
	if ev.Type != "test" || ev.ID != 42 {
		t.Errorf("unexpected event: %+v", ev)
	}
}

func TestSendEvent_MarshalError(t *testing.T) {
	w := httptest.NewRecorder()

	// channels can't be marshaled
	err := SendEvent(w, w, make(chan int))
	if err == nil {
		t.Fatal("expected marshal error")
	}
}
