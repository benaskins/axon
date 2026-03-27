@AGENTS.md

## Conventions
- Use functional options pattern for all public constructors (`ServerOption`, `AuthClientOption`, etc.)
- OTel metrics are auto-wired via `ListenAndServe` — do not add manual metric middleware
- `StandardMiddleware` is deprecated; `ListenAndServe` applies it automatically
- `HealthHandler` is deprecated; use `WithHealthCheck` option instead
- Schema-per-test isolation via `OpenTestDB` — never share test databases

## Constraints
- Never add dependencies on any axon-* module — axon is the foundation layer
- Breaking changes cascade to every service in the workspace; prefer additive changes
- Never expose provider-specific types — this is a generic HTTP toolkit
- `responseWriter` is intentionally unexported; do not promote it

## Testing
- `go test ./...` covers root, `sse/`, and `stream/` packages
- Stream matcher tests use deterministic token sequences — no network required
- Database tests require `AXON_TEST_DSN` or are skipped automatically
