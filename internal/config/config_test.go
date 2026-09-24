package config_test

import (
	"edex-ui-go/internal/config"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg == nil {
		t.Fatal("Expected default config to be non-nil")
	}

	if cfg.Shell == "" {
		t.Errorf("Expected shell to have default value, got empty string")
	}

	if cfg.Theme != "tron" {
		t.Errorf("Expected default theme to be 'tron', got %s", cfg.Theme)
	}

	if cfg.TermFontSize <= 0 {
		t.Errorf("Expected termFontSize > 0, got %d", cfg.TermFontSize)
	}
}

func TestGetConfigDir(t *testing.T) {
	dir := config.GetConfigDir()
	if dir == "" {
		t.Errorf("Expected non-empty config directory")
	}
}
