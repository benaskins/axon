package axon

import (
	"database/sql"
	"net/http"
)

// Deprecated: NewServiceMux is superseded by ListenAndServe which auto-wires
// /health and /metrics. Use WithHealthCheck for database health checks.
func NewServiceMux(db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", HealthHandler(db))
	mux.Handle("GET /metrics", MetricsHandler())
	return mux
}
