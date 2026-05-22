package config

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"time"
)

// Config holds the runtime configuration of the gophermart service.
//
// RunAddress is the TCP address (host:port) the HTTP server listens on.
// DatabaseURI is the PostgreSQL connection string in libpq or URL form.
// AccrualSystemAddress is the base URL of the external accrual service.
// JWTSecret is the symmetric key used to sign authentication tokens.
// JWTSecretGenerated is true when JWTSecret was not provided by the caller
// and was generated at startup; callers can warn about ephemeral signing
// keys in that case.
// JWTTTL is the lifetime of issued authentication tokens.
// AccrualWorkers is the number of goroutines polling the accrual system.
// AccrualPollInterval is the interval between polling cycles.
// ShutdownTimeout bounds the time allotted to graceful shutdown.
type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string

	JWTSecret          string
	JWTSecretGenerated bool
	JWTTTL             time.Duration

	AccrualWorkers      int
	AccrualPollInterval time.Duration

	ShutdownTimeout time.Duration
}

// Load parses the configuration from the provided argument list (typically
// os.Args[1:]) and the process environment. It returns an error if the flag
// set cannot be parsed or if mandatory fields are missing.
func Load(args []string) (*Config, error) {
	cfg := &Config{
		JWTSecret:           "",
		JWTTTL:              24 * time.Hour,
		AccrualWorkers:      4,
		AccrualPollInterval: time.Second,
		ShutdownTimeout:     10 * time.Second,
	}

	fs := flag.NewFlagSet("gophermart", flag.ContinueOnError)
	fs.StringVar(&cfg.RunAddress, "a", ":8080", "HTTP server address (host:port)")
	fs.StringVar(&cfg.DatabaseURI, "d", "", "PostgreSQL connection URI")
	fs.StringVar(&cfg.AccrualSystemAddress, "r", "", "Accrual system base URL")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parse flags: %w", err)
	}

	if v, ok := os.LookupEnv("RUN_ADDRESS"); ok && v != "" {
		cfg.RunAddress = v
	}
	if v, ok := os.LookupEnv("DATABASE_URI"); ok && v != "" {
		cfg.DatabaseURI = v
	}
	if v, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); ok && v != "" {
		cfg.AccrualSystemAddress = v
	}
	if v, ok := os.LookupEnv("JWT_SECRET"); ok && v != "" {
		cfg.JWTSecret = v
	}

	if cfg.RunAddress == "" {
		return nil, fmt.Errorf("run address is required (flag -a or RUN_ADDRESS)")
	}
	if cfg.DatabaseURI == "" {
		return nil, fmt.Errorf("database URI is required (flag -d or DATABASE_URI)")
	}
	if cfg.AccrualSystemAddress == "" {
		return nil, fmt.Errorf("accrual system address is required (flag -r or ACCRUAL_SYSTEM_ADDRESS)")
	}
	if cfg.JWTSecret == "" {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			return nil, fmt.Errorf("generate JWT secret: %w", err)
		}
		cfg.JWTSecret = hex.EncodeToString(buf)
		cfg.JWTSecretGenerated = true
	}

	return cfg, nil
}
