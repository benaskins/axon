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
	mux.HandleFunc("GET /api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	handler := axon.AppHandler(mux, axon.HandlerOptions{})

	axon.ListenAndServe("8080", handler,
		axon.WithDrainTimeout(10*time.Second),
		axon.WithShutdownHook(func(ctx context.Context) {
			slog.Info("cleanup complete")
		}),
	)
}
