package axon

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type serverConfig struct {
	shutdownHooks []func(context.Context)
	tlsConfig     *tls.Config
	tlsCert       string
	tlsKey        string
	drainTimeout  time.Duration
	hookTimeout   time.Duration
}

// ServerOption configures ListenAndServe behavior.
type ServerOption func(*serverConfig)

// WithShutdownHook registers a function to call before draining connections.
// Multiple hooks run in registration order. Use for background task cleanup.
func WithShutdownHook(fn func(context.Context)) ServerOption {
	return func(c *serverConfig) {
		c.shutdownHooks = append(c.shutdownHooks, fn)
	}
}

// WithTLSConfig sets the TLS configuration for the server.
// Requires WithTLSCert to provide certificate and key files.
func WithTLSConfig(cfg *tls.Config) ServerOption {
	return func(c *serverConfig) {
		c.tlsConfig = cfg
	}
}

// WithTLSCert sets the TLS certificate and key files for ListenAndServeTLS.
// Must be used together with WithTLSConfig.
func WithTLSCert(certFile, keyFile string) ServerOption {
	return func(c *serverConfig) {
		c.tlsCert = certFile
		c.tlsKey = keyFile
	}
}

// WithDrainTimeout sets the maximum time to wait for in-flight requests
// to complete during shutdown. Defaults to 30 seconds.
func WithDrainTimeout(d time.Duration) ServerOption {
	return func(c *serverConfig) {
		c.drainTimeout = d
	}
}

// WithHookTimeout sets the maximum time to wait for shutdown hooks
// to complete. Defaults to 10 seconds.
func WithHookTimeout(d time.Duration) ServerOption {
	return func(c *serverConfig) {
		c.hookTimeout = d
	}
}

// ListenAndServe starts an HTTP server and blocks until SIGINT or SIGTERM.
// Performs graceful shutdown: runs shutdown hooks, then drains connections.
func ListenAndServe(port string, handler http.Handler, opts ...ServerOption) {
	cfg := &serverConfig{
		drainTimeout: 30 * time.Second,
		hookTimeout:  10 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	srv := &http.Server{
		Addr:             ":" + port,
		Handler:          handler,
		TLSConfig:        cfg.tlsConfig,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("service starting", "port", port, "tls", cfg.tlsConfig != nil)
		var err error
		if cfg.tlsConfig != nil && cfg.tlsCert != "" {
			err = srv.ListenAndServeTLS(cfg.tlsCert, cfg.tlsKey)
		} else {
			err = srv.ListenAndServe()
		}
		if err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")

	// Phase 1: run shutdown hooks with their own timeout
	hookCtx, hookCancel := context.WithTimeout(context.Background(), cfg.hookTimeout)
	defer hookCancel()

	for _, hook := range cfg.shutdownHooks {
		hook(hookCtx)
	}

	// Phase 2: drain in-flight requests
	drainCtx, drainCancel := context.WithTimeout(context.Background(), cfg.drainTimeout)
	defer drainCancel()

	if err := srv.Shutdown(drainCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}

	slog.Info("shutdown complete")
}
