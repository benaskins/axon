package axon_test

import (
	"net/http"
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
