package axon

import (
	"net/http"

	"github.com/justinas/alice"
)

// StandardMiddleware wraps a handler with the standard middleware stack:
// MetaHeaders (outermost) -> RequestLogging -> RequestMetrics -> handler
//
// MetaHeaders runs first so that X-Axon-Run-Id and X-Axon-Trace-Id are
// available in context for downstream logging and metrics middleware.
func StandardMiddleware(handler http.Handler) http.Handler {
	chain := alice.New(
		MetaHeaders,
		RequestLogging,
		RequestMetrics,
	)

	return chain.Then(handler)
}
