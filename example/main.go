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

	axon.ListenAndServe("8080", mux,
		axon.WithHealthCheck("example", func() error { return nil }),
		axon.WithDrainTimeout(10*time.Second),
		axon.WithShutdownHook(func(ctx context.Context) {
			slog.Info("cleanup complete")
		}),
	)
}
