package axon_test

import (
	"testing"

	"github.com/benaskins/axon"
)

func TestMustLoadConfig_Defaults(t *testing.T) {
	type Config struct {
		Port string `env:"TEST_AXON_PORT" envDefault:"9999"`
	}
	var cfg Config
	axon.MustLoadConfig(&cfg)
	if cfg.Port != "9999" {
		t.Errorf("expected default port 9999, got %s", cfg.Port)
	}
}

func TestMustLoadConfig_FromEnv(t *testing.T) {
	t.Setenv("TEST_AXON_HOST", "example.com")
	type Config struct {
		Host string `env:"TEST_AXON_HOST"`
	}
	var cfg Config
	axon.MustLoadConfig(&cfg)
	if cfg.Host != "example.com" {
		t.Errorf("expected example.com, got %s", cfg.Host)
	}
}
