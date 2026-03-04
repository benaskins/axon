package axon

import (
	"net/http"

	"github.com/justinas/alice"
)

// StandardMiddleware wraps a handler with the standard middleware stack:
// RequestLogging -> RequestMetrics -> handler
func StandardMiddleware(handler http.Handler) http.Handler {
	chain := alice.New(
		MetaHeaders,
		RequestLogging,
		RequestMetrics,
	)

	return chain.Then(handler)
}
