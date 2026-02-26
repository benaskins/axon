package axon

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// HealthHandler returns a health check handler.
// If db is non-nil, includes database connectivity status.
func HealthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"status": "healthy"}

		if db != nil {
			if err := db.Ping(); err != nil {
				resp["database"] = "disconnected"
			} else {
				resp["database"] = "connected"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
