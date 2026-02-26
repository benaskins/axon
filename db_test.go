package axon_test

import (
	"testing"

	"github.com/benaskins/axon"
)

func TestOpenDB_BadDSN(t *testing.T) {
	_, err := axon.OpenDB("postgres://invalid:invalid@localhost:59999/nope?sslmode=disable&connect_timeout=1", "test")
	if err == nil {
		t.Error("expected error for bad DSN")
	}
}
