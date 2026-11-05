package config

import "testing"

func TestConfigInit(t *testing.T) {
	cfg := Config{}
	if cfg.Verbose || cfg.NoColor || cfg.JSON {
		t.Errorf("expected default Config fields to be false, got %+v", cfg)
	}

	cfg.Verbose = true
	cfg.NoColor = true
	cfg.JSON = true
	if !cfg.Verbose || !cfg.NoColor || !cfg.JSON {
		t.Errorf("failed to set Config flags, got %+v", cfg)
	}
}
