package axon

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// LoadConfig parses environment variables into the provided struct.
// The struct should use `env` and `envDefault` tags from caarlos0/env.
func LoadConfig(cfg any) error {
	if err := env.Parse(cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	return nil
}

// MustLoadConfig parses environment variables into the provided struct.
// Panics on failure.
func MustLoadConfig(cfg any) {
	if err := LoadConfig(cfg); err != nil {
		panic(fmt.Sprintf("axon: %v", err))
	}
}
