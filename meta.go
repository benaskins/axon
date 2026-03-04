package axon

import (
	"context"
	"net/http"
	"strings"
)

const (
	// MetaKey is the context key for X-Axon-* header metadata.
	MetaKey contextKey = "axon_meta"

	headerPrefix = "X-Axon-"
)

// MetaHeaders returns middleware that extracts X-Axon-* headers into the
// request context. Downstream handlers access values via Meta or RunID.
//
// Header names are normalised to lowercase keys with the prefix stripped:
// X-Axon-Run-Id → "run-id", X-Axon-Trace-Id → "trace-id".
func MetaHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta := make(map[string]string)
		for name, values := range r.Header {
			if strings.HasPrefix(name, headerPrefix) && len(values) > 0 {
				// Normalise Go's canonical "Run-Id" to "run-id"
			key := strings.ToLower(name[len(headerPrefix):])
				meta[key] = values[0]
			}
		}

		if len(meta) > 0 {
			ctx := context.WithValue(r.Context(), MetaKey, meta)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// Meta retrieves an X-Axon-* header value from the request context.
// The key should be lowercase with prefix stripped: "run-id", "trace-id".
// Returns empty string if not present.
func Meta(ctx context.Context, key string) string {
	meta, _ := ctx.Value(MetaKey).(map[string]string)
	return meta[key]
}

// RunID is a shortcut for Meta(ctx, "run-id").
func RunID(ctx context.Context) string {
	return Meta(ctx, "run-id")
}
