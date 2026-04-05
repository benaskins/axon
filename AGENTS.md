---
module: github.com/benaskins/axon
kind: library
---

# axon

This file provides guidance when working with code in this repository.

## Build & Test Commands

```bash
go test ./...              # Run all tests (3 packages)
go test .                  # Root package tests only
go test ./sse              # SSE package tests only
go test ./stream           # Stream package tests only
go test -run TestName ./   # Run a single test
go test -v ./...           # Verbose output
go fmt ./...               # Format
go vet ./...               # Lint
```

No Makefile -- standard Go tooling only. Go 1.26.1.

## Architecture

Axon is a Go toolkit for building AI-powered web services. Single module (`github.com/benaskins/axon`) with three packages:

### Root package (`axon`)
Core HTTP service building blocks:
- **Server lifecycle** (`server.go`) -- `ListenAndServe` with graceful shutdown (SIGINT/SIGTERM), functional options (`WithShutdownHook`, `WithDrainTimeout`, `WithHookTimeout`, `WithTLSConfig`). Shutdown runs hooks (with hookTimeout) then drains connections (with drainTimeout).
- **Auth** (`auth.go`, `auth_middleware.go`) -- `SessionValidator` interface with `AuthClient` implementation. `AuthClient` validates sessions against a remote service, configurable via `AuthClientOption`: `WithEndpointPath`, `WithTokenSender`, `WithDecodeFunc`, `WithCacheTTL`. `NewAuthClient` (mTLS, returns error) and `NewAuthClientPlain` (plain HTTP). `RequireAuth` middleware accepts `SessionValidator`, extracts `SessionInfo` into context. Options: `WithCookieName`, `WithTokenExtractor`.
- **SessionInfo** -- Claims-based: `SessionInfo.Claims` map with `UserID()`, `Username()`, `Claim(key)` accessors. Context helpers: `UserID(ctx)`, `Username(ctx)`, `Session(ctx)`.
- **Config** (`config.go`) -- `MustLoadConfig` parses env vars via `caarlos0/env` struct tags
- **Middleware** (`middleware.go`) -- `StandardMiddleware` chains logging + OTel/Prometheus metrics via alice. Deprecated: applied automatically by `ListenAndServe`. Metrics use `r.Pattern` (Go 1.22+) to avoid high-cardinality labels.
- **Metrics** (`metrics.go`) -- OTel instruments (histogram, counter) exported as Prometheus metrics. `MeterProvider()` lets domain packages create their own meters. `MetricsHandler()` serves Prometheus exposition format.
- **Handler wrapping** (`wrap.go`) -- `WrapHandler` auto-wires `/health` and `/metrics` routes. `WithHealthCheck` registers named health checks with `ListenAndServe`.
- **Meta headers** (`meta.go`) -- `MetaHeaders` middleware extracts `X-Axon-*` headers into context. `Meta(ctx, key)` and `RunID(ctx)` accessors.
- **Request decoding** (`request.go`) -- `DecodeJSON[T]` decodes + validates request bodies (1MB limit). `Validatable` interface for auto-validation.
- **HTTP client** (`client.go`) -- `StatusError` type for unexpected HTTP status codes. `IsStatusError` helper.
- **Errors** (`errors.go`) -- Sentinel errors: `ErrUnauthorized`, `ErrNotFound`, `ErrServiceUnavailable`.
- **SPA** (`spa.go`) -- `SPAHandler(files, subdir, opts...)` serves embedded static files with client-side routing fallback. Use `WithStaticPrefix(prefix)` to 404 on missing assets under that prefix instead of falling back to index.html.
- **Helpers** -- `WriteJSON`/`WriteError` (response.go, logs encoding errors), `ValidateSlug` (slug.go), `HealthHandler` (health.go, deprecated -- use `WithHealthCheck`)

### `sse/` -- Server-Sent Events
- `SetSSEHeaders`/`SendEvent` -- SSE protocol helpers
- `EventBus[T]` -- Generic in-memory pub/sub with buffered channels; copies subscriber map before sending to avoid lock contention

### `stream/` -- AI Model Stream Filtering
- `StreamFilter` -- Buffered token filter with lookahead; feeds tokens through matchers before emitting
- `ToolCallMatcher` -- Detects JSON tool calls in streamed text (objects, arrays, fenced code blocks)
- `ContentSafetyMatcher` -- Regex-based blocked pattern detection with cross-boundary overlap

## Key Patterns

- **Functional options** for configuration: `ServerOption`, `AuthClientOption`, `AuthOption`, `SPAOption`
- **SessionValidator interface** for auth: implement `ValidateSession(token string) (*SessionInfo, error)`
- **Matcher interface** in stream package: `Scan(buf, prevTail) MatchResult` + optional `Extractable` for data extraction
- **Context keys** for auth: `UserID(ctx)`, `Username(ctx)`, `Session(ctx)` for full `*SessionInfo`
- **`embed.FS`** for SPA static files
- **responseWriter** (unexported) wraps http.ResponseWriter to capture status codes for logging/metrics

## Dependencies

caarlos0/env (config), justinas/alice (middleware chaining), jackc/pgx (postgres, used by health.go and mux.go), OTel SDK + prometheus exporter (metrics). Database management (pool, migrations) moved to axon-base.
