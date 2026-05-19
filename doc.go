// Package axon provides a Go toolkit for building AI-powered web services.
//
// It includes HTTP server lifecycle, configuration, database management,
// health checks, OpenTelemetry request metrics, request logging, auth
// middleware, SPA static file serving, and slug validation.
//
// AppHandler is the main entry point for wrapping a user handler with
// observability. ListenAndServe handles the long-running server lifecycle;
// callers running on FaaS adapters can skip it and mount the wrapped
// handler directly.
package axon
