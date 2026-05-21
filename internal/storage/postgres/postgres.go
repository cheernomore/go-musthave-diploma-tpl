package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB owns a PostgreSQL connection pool and acts as a factory for the
// repositories used by the gophermart service. The underlying pool is not
// exposed; callers obtain typed repositories instead.
type DB struct {
	pool *pgxpool.Pool
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

	return &DB{pool: pool}, nil
}

// Users returns a repository for user records.
func (d *DB) Users() *UserRepository { return NewUserRepository(d.pool) }

// Orders returns a repository for orders and accrual processing operations.
func (d *DB) Orders() *OrderRepository { return NewOrderRepository(d.pool) }

// Balances returns a repository for balances and withdrawals.
func (d *DB) Balances() *BalanceRepository { return NewBalanceRepository(d.pool) }

// Close releases all resources held by the pool.
func (d *DB) Close() {
	if d != nil && d.pool != nil {
		d.pool.Close()
	}
}
