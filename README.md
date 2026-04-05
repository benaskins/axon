# axon

> Toolkit · Part of the [lamina](https://github.com/benaskins/lamina-mono) workspace

Go toolkit for building LLM-powered web services. Provides HTTP server lifecycle, auth, metrics, SSE streaming, and token stream filtering. Each package can be used independently. Database management is handled by [axon-base](https://github.com/benaskins/axon-base).

## Getting started

```
go get github.com/benaskins/axon@latest
```

A minimal service with health checks and graceful shutdown:

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
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		axon.WriteJSON(w, http.StatusOK, map[string]string{"msg": "hello"})
	})

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

**Database** — Use [axon-base](https://github.com/benaskins/axon-base) for PostgreSQL pool, migrations, and repository patterns.

```go
p, _ := pool.NewPool(ctx, dsn, "myapp")
db, _ := p.StdDB()
migration.Run(db, migrationsFS, "migrations")
```

**SPA** — Serve embedded static files with client-side routing fallback.

```go
mux.Handle("/", axon.SPAHandler(staticFS, "build",
    axon.WithStaticPrefix("/_app/"),
))
```

**Health checks** — `ListenAndServe` auto-wires `/health` and `/metrics`. Add named checks with `WithHealthCheck`:

```go
axon.ListenAndServe("8080", mux,
    axon.WithHealthCheck("database", func() error { return db.Ping() }),
)
```

**Config and helpers:**

```go
axon.MustLoadConfig(&cfg)               // env vars -> struct
axon.DecodeJSON[MyRequest](w, r)        // decode + validate request body
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
