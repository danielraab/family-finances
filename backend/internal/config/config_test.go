package config_test

import (
	"testing"

	"at.draab/familyfinances/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")

	cfg := config.Load()

	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "8080")
	}
}

func TestLoadPortOverride(t *testing.T) {
	t.Setenv("PORT", "9999")

	cfg := config.Load()

	if cfg.Port != "9999" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "9999")
	}
}
