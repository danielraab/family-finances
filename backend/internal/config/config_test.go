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

func TestLoadDatabaseURLEmptyWhenUnset(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	cfg := config.Load()

	if cfg.DatabaseURL != "" {
		t.Fatalf("DatabaseURL = %q, want empty", cfg.DatabaseURL)
	}
}

func TestLoadDatabaseURLFromEnv(t *testing.T) {
	const dsn = "postgres://u:p@localhost:5432/family_finances?sslmode=disable"
	t.Setenv("DATABASE_URL", dsn)

	cfg := config.Load()

	if cfg.DatabaseURL != dsn {
		t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, dsn)
	}
}
