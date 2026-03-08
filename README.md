# axon

> Toolkit · Part of the [lamina](https://github.com/benaskins/lamina-mono) workspace

Go toolkit for building LLM-powered web services. Provides HTTP server lifecycle, auth, database management, metrics, SSE streaming, and token stream filtering. Each package can be used independently.

## Getting started

```
go get github.com/benaskins/axon@latest
```

A minimal service with a health endpoint and graceful shutdown:

```go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/benaskins/axon"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", axon.HealthHandler(nil))

	axon.ListenAndServe("8080", mux,
		axon.WithDrainTimeout(10*time.Second),
		axon.WithShutdownHook(func(ctx context.Context) {
			slog.Info("cleanup complete")
		}),
	)
}
```

## Packages

### `axon` — Core service toolkit

**Server lifecycle** — `ListenAndServe` with graceful shutdown (SIGINT/SIGTERM), shutdown hooks, configurable drain and hook timeouts.

```go
axon.ListenAndServe("8080", mux,
    axon.WithShutdownHook(cleanup),
    axon.WithDrainTimeout(30*time.Second),
)
```

**Auth** — `SessionValidator` interface with `AuthClient` implementation. Validates sessions against a remote auth service with caching.

```go
client := axon.NewAuthClientPlain(authURL,
    axon.WithEndpointPath("/auth/check"),
    axon.WithCacheTTL(time.Minute),
)

mux.Handle("/api/", axon.RequireAuth(client)(apiHandler))

// In handlers:
userID := axon.UserID(r.Context())
session := axon.Session(r.Context())
role := session.Claim("role")
```

**Database** — PostgreSQL with schema isolation, goose migrations from embedded FS, and per-test schemas.

```go
db := axon.MustOpenDB(dsn, "myapp")
axon.MustRunMigrations(db, migrationsFS)

// In tests:
db := axon.OpenTestDB(t, dsn, migrationsFS)
```

**SPA** — Serve embedded static files with client-side routing fallback.

```go
mux.Handle("/", axon.SPAHandler(staticFS, "build",
    axon.WithStaticPrefix("/_app/"),
))
```

**Middleware, config, and helpers:**

```go
handler := axon.StandardMiddleware(mux) // logging + Prometheus metrics
axon.MustLoadConfig(&cfg)               // env vars -> struct
mux.HandleFunc("/health", axon.HealthHandler(db))
```

### `sse/` — Server-Sent Events

```go
sse.SetSSEHeaders(w)
sse.SendEvent(w, flusher, payload)

bus := sse.NewEventBus[Event]()
ch := bus.Subscribe("client-1")
bus.Publish(Event{Type: "update"})
```

### `stream/` — LLM stream filtering

Buffered token filter with lookahead for processing LLM output in real time.

```go
filter := stream.NewStreamFilter(
    stream.NewToolCallMatcher(),
    stream.NewContentSafetyMatcher(blockedPatterns),
)

for token := range tokens {
    emitted, held := filter.Process(token)
    // emitted: safe to send to client
    // held: buffered pending matcher decisions
}
```

## License

MIT
