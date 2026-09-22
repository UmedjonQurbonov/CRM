package config

import (
	"os"
	"testing"
)

func TestConfigLoadDefaults(t *testing.T) {
	// Clear any existing env vars for deterministic testing
	_ = os.Unsetenv("SERVER_PORT")
	_ = os.Unsetenv("DB_PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	if cfg.Server.Port != "8080" {
		t.Errorf("expected default server port 8080, got %s", cfg.Server.Port)
	}

	if cfg.Database.Port != "5433" {
		t.Errorf("expected default DB port 5433, got %s", cfg.Database.Port)
	}

	expectedDSN := "postgres://postgres:postgrespassword@localhost:5433/crm_db?sslmode=disable"
	if cfg.Database.DSN() != expectedDSN {
		t.Errorf("expected DSN %q, got %q", expectedDSN, cfg.Database.DSN())
	}
}

func TestConfigLoadCustomEnv(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_NAME", "custom_crm")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	if cfg.Server.Port != "9090" {
		t.Errorf("expected custom server port 9090, got %s", cfg.Server.Port)
	}

	if cfg.Database.Port != "5432" {
		t.Errorf("expected custom DB port 5432, got %s", cfg.Database.Port)
	}

	if cfg.Database.Name != "custom_crm" {
		t.Errorf("expected custom DB name custom_crm, got %s", cfg.Database.Name)
	}
}
