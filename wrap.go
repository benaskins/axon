package axon

import (
	"encoding/json"
	"net/http"
)

// HealthCheck is a named health check function.
type HealthCheck struct {
	Name  string
	Check func() error
}

// WithHealthCheck registers a named health check with ListenAndServe.
// When GET /health is called, all checks run and their results are included
// in the response. If any check fails, the endpoint returns 503.
func WithHealthCheck(name string, check func() error) ServerOption {
	return func(c *serverConfig) {
		c.healthChecks = append(c.healthChecks, HealthCheck{Name: name, Check: check})
	}
}

// WrapHandler wraps a user handler with automatic observability:
//   - GET /metrics — Prometheus metrics endpoint
//   - GET /health  — health check endpoint (extensible via WithHealthCheck)
//   - All other routes — delegated through StandardMiddleware (logging + metrics)
//
// Routes /metrics and /health are reserved and cannot be overridden by the
// inner handler.
func WrapHandler(handler http.Handler, checks ...HealthCheck) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /metrics", MetricsHandler())
	mux.HandleFunc("GET /health", healthHandler(checks))

	// Everything else goes through StandardMiddleware then to the user handler.
	mux.Handle("/", StandardMiddleware(handler))

	return mux
}

func healthHandler(checks []HealthCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if len(checks) == 0 {
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}

		status := http.StatusOK
		result := map[string]any{"status": "ok"}
		checkResults := make(map[string]string, len(checks))

		for _, hc := range checks {
			if err := hc.Check(); err != nil {
				checkResults[hc.Name] = err.Error()
				status = http.StatusServiceUnavailable
				result["status"] = "unhealthy"
			} else {
				checkResults[hc.Name] = "ok"
			}
		}

		result["checks"] = checkResults
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(result)
	}
}
