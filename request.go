package axon

import (
	"encoding/json"
	"net/http"
)

// Validatable is implemented by request types that can validate themselves.
// DecodeJSON calls Validate automatically after successful decoding.
type Validatable interface {
	Validate() error
}

// DecodeJSON decodes the request body as JSON into T.
// The body is limited to 1MB to prevent abuse.
// If T implements Validatable, validation runs after decoding.
// On failure, it writes an error response and returns (zero, false).
// On success, it returns (value, true).
func DecodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var v T
	if r.Body == nil {
		WriteError(w, http.StatusBadRequest, "request body is required")
		return v, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return v, false
	}
	if val, ok := any(v).(Validatable); ok {
		if err := val.Validate(); err != nil {
			WriteError(w, http.StatusUnprocessableEntity, err.Error())
			return v, false
		}
	}
	return v, true
}
