package axon

import (
	"database/sql"
	"net/http"
)

// NewServiceMux returns an http.ServeMux pre-wired with /health and /metrics
// endpoints. Services add their own routes to the returned mux.
func NewServiceMux(db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", HealthHandler(db))
	mux.Handle("GET /metrics", MetricsHandler())
	return mux
}
