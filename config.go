package axon

import (
	"log/slog"
	"os"

	"github.com/caarlos0/env/v11"
)

// MustLoadConfig parses environment variables into the provided struct.
// The struct should use `env` and `envDefault` tags from caarlos0/env.
// Exits the process if parsing fails.
func MustLoadConfig(cfg any) {
	if err := env.Parse(cfg); err != nil {
		slog.Error("failed to parse config", "error", err)
		os.Exit(1)
	}
}
