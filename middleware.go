package axon

import (
	"net/http"

	"github.com/justinas/alice"
)

// Deprecated: StandardMiddleware is applied automatically by ListenAndServe.
// Only use directly if serving HTTP without ListenAndServe.
func StandardMiddleware(handler http.Handler) http.Handler {
	chain := alice.New(
		MetaHeaders,
		RequestLogging,
		RequestMetrics,
	)

	return chain.Then(handler)
}
