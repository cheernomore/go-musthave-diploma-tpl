package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is a thin wrapper around a pgx connection pool. It owns the lifecycle of
// the pool and exposes it to repositories.
type DB struct {
	Pool *pgxpool.Pool
}

// New opens a connection pool to the database described by databaseURI and
// verifies connectivity with a Ping. Schema migrations are applied
// automatically. The returned *DB must be closed by the caller.
func New(ctx context.Context, databaseURI string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(databaseURI)
	if err != nil {
		return nil, fmt.Errorf("parse database URI: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	if err := Migrate(databaseURI); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close releases all resources held by the pool.
func (d *DB) Close() {
	if d != nil && d.Pool != nil {
		d.Pool.Close()
	}
}
