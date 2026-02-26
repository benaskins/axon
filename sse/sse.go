package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SetSSEHeaders configures the response for Server-Sent Events streaming.
func SetSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
}

// SendEvent marshals data as JSON and writes it as an SSE data frame.
func SendEvent(w http.ResponseWriter, flusher http.Flusher, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "data: %s\n\n", b)
	flusher.Flush()
	return nil
}
