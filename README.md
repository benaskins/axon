# axon

A Go toolkit for building AI-powered web services.

Axon provides the common building blocks for HTTP services that work with AI models: server lifecycle, auth, database management, metrics, SSE streaming, and token stream filtering.

## Install

```
go get github.com/benaskins/axon@latest
```

Requires Go 1.24+.

## Packages

### `axon` — Core service toolkit

**Server lifecycle** — Graceful shutdown with SIGINT/SIGTERM, shutdown hooks, configurable drain and hook timeouts.

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
axon.MustLoadConfig(&cfg)               // env vars → struct
mux.HandleFunc("/health", axon.HealthHandler(db))
```

### `sse` — Server-Sent Events

```go
sse.SetSSEHeaders(w)
sse.SendEvent(w, flusher, payload)

bus := sse.NewEventBus[Event]()
ch := bus.Subscribe("client-1")
bus.Publish(Event{Type: "update"})
```

### `stream` — AI model stream filtering

Buffered token filter with lookahead for processing AI model output in real time.

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

## CI

```bash
# Install tools (one-time)
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
go install golang.org/x/vuln/cmd/govulncheck@latest

# Run all checks
./scripts/ci.sh
```

Runs: build, vet, test (race detector), gosec, staticcheck, govulncheck.

## License

Apache 2.0 — see [LICENSE](LICENSE).
