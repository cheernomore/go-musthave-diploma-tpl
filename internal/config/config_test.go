package config

import (
	"testing"
)

func TestLoadFlags(t *testing.T) {
	t.Setenv("RUN_ADDRESS", "")
	t.Setenv("DATABASE_URI", "")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "")

	cfg, err := Load([]string{
		"-a", ":9090",
		"-d", "postgres://u:p@localhost/db",
		"-r", "http://accrual:8081",
	})
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

	cfg, err := Load([]string{
		"-a", ":9090",
		"-d", "postgres://flag/db",
		"-r", "http://flag-accrual",
	})
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

	if _, err := Load([]string{"-a", ":8080"}); err == nil {
		t.Fatal("expected error for missing required fields")
	}
}
