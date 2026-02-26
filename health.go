package axon

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// HealthHandler returns a health check handler.
// If db is non-nil, includes database connectivity status.
// Returns 503 when the database ping fails.
func HealthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"status": "healthy"}
		status := http.StatusOK

		if db != nil {
			if err := db.Ping(); err != nil {
				resp["status"] = "unhealthy"
				resp["database"] = "disconnected"
				status = http.StatusServiceUnavailable
			} else {
				resp["database"] = "connected"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(resp)
	}
}
