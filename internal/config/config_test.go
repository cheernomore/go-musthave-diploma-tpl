package config

import (
	"io"
	"log/slog"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestLoadFlags(t *testing.T) {
	t.Setenv("RUN_ADDRESS", "")
	t.Setenv("DATABASE_URI", "")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "")
	t.Setenv("JWT_SECRET", "test-secret")

	cfg, err := Load([]string{
		"-a", ":9090",
		"-d", "postgres://u:p@localhost/db",
		"-r", "http://accrual:8081",
	}, discardLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RunAddress != ":9090" {
		t.Errorf("RunAddress = %q", cfg.RunAddress)
	}
	if cfg.DatabaseURI != "postgres://u:p@localhost/db" {
		t.Errorf("DatabaseURI = %q", cfg.DatabaseURI)
	}
	if cfg.AccrualSystemAddress != "http://accrual:8081" {
		t.Errorf("AccrualSystemAddress = %q", cfg.AccrualSystemAddress)
	}
}

func TestLoadEnvOverridesFlags(t *testing.T) {
	t.Setenv("RUN_ADDRESS", ":7070")
	t.Setenv("DATABASE_URI", "postgres://env/db")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://env-accrual")
	t.Setenv("JWT_SECRET", "test-secret")

	cfg, err := Load([]string{
		"-a", ":9090",
		"-d", "postgres://flag/db",
		"-r", "http://flag-accrual",
	}, discardLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RunAddress != ":7070" {
		t.Errorf("RunAddress = %q", cfg.RunAddress)
	}
	if cfg.DatabaseURI != "postgres://env/db" {
		t.Errorf("DatabaseURI = %q", cfg.DatabaseURI)
	}
	if cfg.AccrualSystemAddress != "http://env-accrual" {
		t.Errorf("AccrualSystemAddress = %q", cfg.AccrualSystemAddress)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	t.Setenv("RUN_ADDRESS", "")
	t.Setenv("DATABASE_URI", "")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "")
	t.Setenv("JWT_SECRET", "test-secret")

	if _, err := Load([]string{"-a", ":8080"}, discardLogger()); err == nil {
		t.Fatal("expected error for missing required fields")
	}
}

func TestLoadGeneratesJWTSecretWhenMissing(t *testing.T) {
	t.Setenv("RUN_ADDRESS", "")
	t.Setenv("DATABASE_URI", "")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "")
	t.Setenv("JWT_SECRET", "")

	cfg, err := Load([]string{
		"-a", ":9090",
		"-d", "postgres://u:p@localhost/db",
		"-r", "http://accrual:8081",
	}, discardLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 32 random bytes hex-encoded → exactly 64 characters.
	if got := len(cfg.JWTSecret); got != 64 {
		t.Fatalf("generated secret length = %d, want 64", got)
	}
}
