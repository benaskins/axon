package axon

import (
	"encoding/json"
	"net/http"
)

// DecodeJSON decodes the request body as JSON into T.
// On failure, it writes a 400 error response and returns (zero, false).
// On success, it returns (value, true).
func DecodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var v T
	if r.Body == nil {
		WriteError(w, http.StatusBadRequest, "request body is required")
		return v, false
	}
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return v, false
	}
	return v, true
}
