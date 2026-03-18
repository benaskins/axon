package axon

import (
	"encoding/json"
	"net/http"
)

// WrapHandler wraps a user handler with automatic observability:
//   - GET /metrics — Prometheus metrics endpoint
//   - GET /health  — health check endpoint (always 200, extensible via WithHealthCheck)
//   - All other routes — delegated through StandardMiddleware (logging + metrics)
//
// Routes /metrics and /health are reserved and cannot be overridden by the
// inner handler.
func WrapHandler(handler http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /metrics", MetricsHandler())
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Everything else goes through StandardMiddleware then to the user handler.
	mux.Handle("/", StandardMiddleware(handler))

	return mux
}
