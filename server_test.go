package axon_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/benaskins/axon"
)

func TestListenAndServe_ShutdownOnSignal(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	done := make(chan struct{})
	go func() {
		axon.ListenAndServe("0", mux) // port 0 = random available port
		close(done)
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Send SIGINT to trigger shutdown
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	select {
	case <-done:
		// Success -- server shut down
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down within 5 seconds")
	}
}

func TestListenAndServe_ShutdownHookRuns(t *testing.T) {
	var hookCalled atomic.Bool

	done := make(chan struct{})
	go func() {
		axon.ListenAndServe("0", http.NewServeMux(),
			axon.WithShutdownHook(func(ctx context.Context) {
				hookCalled.Store(true)
			}),
		)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down within 5 seconds")
	}

	if !hookCalled.Load() {
		t.Error("shutdown hook was not called")
	}
}

func TestListenAndServe_MultipleShutdownHooks(t *testing.T) {
	var order []int
	orderCh := make(chan []int, 1)

	done := make(chan struct{})
	go func() {
		axon.ListenAndServe("0", http.NewServeMux(),
			axon.WithShutdownHook(func(ctx context.Context) {
				order = append(order, 1)
			}),
			axon.WithShutdownHook(func(ctx context.Context) {
				order = append(order, 2)
				orderCh <- order
			}),
		)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down within 5 seconds")
	}

	result := <-orderCh
	if len(result) != 2 || result[0] != 1 || result[1] != 2 {
		t.Errorf("hooks ran in wrong order: %v", result)
	}
}

func TestWithDrainTimeout(t *testing.T) {
	// Just verify the option can be applied without error
	done := make(chan struct{})
	go func() {
		axon.ListenAndServe("0", http.NewServeMux(),
			axon.WithDrainTimeout(5*time.Second),
		)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down within 5 seconds")
	}
}
